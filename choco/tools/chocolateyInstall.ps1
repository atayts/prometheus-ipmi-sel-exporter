$ErrorActionPreference = 'Stop'

$serviceName = 'ipmi_sel_win_exporter'
$installDir  = Join-Path $env:ProgramFiles 'prometheus-ipmi-exporter'
$exeName     = 'ipmi_sel_win_exporter.exe'
$exePath     = Join-Path $installDir $exeName
$configName  = 'config.yml'
$configPath  = Join-Path $installDir $configName
$toolsDir    = Split-Path -Parent $MyInvocation.MyCommand.Definition

# Stop and remove any existing service FIRST. A running service holds an open
# handle on the executable, so copying the new binary over it fails with
# "the process cannot access the file ... because it is being used by another
# process". Nothing below may touch the install directory until this is done.
$svc = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($svc) {
    if ($svc.Status -ne 'Stopped') {
        Write-Host "Stopping existing $serviceName service..."
        Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        $svc.WaitForStatus('Stopped', '00:00:30')
    }
    Write-Host "Removing existing $serviceName service..."
    sc.exe delete $serviceName | Out-Null
    Start-Sleep -Seconds 2
}

# Windows can keep the image handle open for a moment after the service exits,
# so give the copy a few tries before giving up.
function Copy-WithRetry {
    param([string]$Source, [string]$Destination, [int]$Attempts = 10)

    for ($i = 1; $i -le $Attempts; $i++) {
        try {
            Copy-Item $Source -Destination $Destination -Force
            return
        } catch {
            if ($i -eq $Attempts) { throw }
            Write-Host "  $Destination is locked, retrying ($i/$Attempts)..."
            Start-Sleep -Seconds 2
        }
    }
}

# Create install directory.
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Copy the binary from the package tools directory.
Copy-WithRetry (Join-Path $toolsDir $exeName) $exePath

# The exporter reads config.yml from its own directory. Never overwrite an
# existing one, it holds the site's settings and must survive upgrades. The
# packaged defaults are always dropped alongside it as a reference.
Copy-Item (Join-Path $toolsDir $configName) -Destination "$configPath.example" -Force
if (Test-Path $configPath) {
    Write-Host "Keeping existing configuration at $configPath"
} else {
    Copy-Item (Join-Path $toolsDir $configName) -Destination $configPath -Force
    Write-Host "Wrote default configuration to $configPath"
}

# Create and start the Windows service. All options come from config.yml, so
# the binPath carries no arguments and never needs to change.
Write-Host "Creating $serviceName service..."
sc.exe create $serviceName `
    binPath= "`"$exePath`"" `
    start= auto `
    DisplayName= "Prometheus IPMI Exporter" | Out-Null

sc.exe description $serviceName "Prometheus exporter for IPMI System Event Log (SEL) hardware alerts" | Out-Null

Write-Host "Starting $serviceName service..."
Start-Service -Name $serviceName

Write-Host ""
Write-Host "Prometheus IPMI Exporter installed and running."
Write-Host "To change any option, edit $configPath and run:"
Write-Host "    Restart-Service $serviceName"
Write-Host "Startup and scrape errors are logged to $(Join-Path $installDir "$serviceName.log")"
