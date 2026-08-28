$ErrorActionPreference = 'Stop'

$serviceName = 'ipmi_sel_win_exporter'
$installDir  = Join-Path $env:ProgramFiles 'prometheus-ipmi-exporter'
$exeName     = 'ipmi_sel_win_exporter.exe'
$exePath     = Join-Path $installDir $exeName
$configName  = 'config.yml'
$configPath  = Join-Path $installDir $configName

# Create install directory.
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Copy the binary from the package tools directory.
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
Copy-Item (Join-Path $toolsDir $exeName) -Destination $exePath -Force

# The exporter reads config.yml from its own directory. Never overwrite an
# existing one — it holds the site's settings and must survive upgrades. The
# packaged defaults are always dropped alongside it as a reference.
Copy-Item (Join-Path $toolsDir $configName) -Destination "$configPath.example" -Force
if (Test-Path $configPath) {
    Write-Host "Keeping existing configuration at $configPath"
} else {
    Copy-Item (Join-Path $toolsDir $configName) -Destination $configPath -Force
    Write-Host "Wrote default configuration to $configPath"
}

# Stop existing service if running (upgrade scenario).
$svc = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($svc) {
    if ($svc.Status -eq 'Running') {
        Write-Host "Stopping existing $serviceName service..."
        Stop-Service -Name $serviceName -Force
        # Wait for the service to fully stop before replacing the binary.
        $svc.WaitForStatus('Stopped', '00:00:30')
    }
    Write-Host "Removing existing $serviceName service..."
    sc.exe delete $serviceName | Out-Null
    Start-Sleep -Seconds 2
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
