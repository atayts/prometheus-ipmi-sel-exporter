$ErrorActionPreference = 'Stop'

$serviceName = 'ipmi_sel_win_exporter'
$installDir  = Join-Path $env:ProgramFiles 'prometheus-ipmi-exporter'
$exeName     = 'ipmi_sel_win_exporter.exe'
$exePath     = Join-Path $installDir $exeName

# Create install directory.
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Copy the binary from the package tools directory.
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
Copy-Item (Join-Path $toolsDir $exeName) -Destination $exePath -Force

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

# Create and start the Windows service.
Write-Host "Creating $serviceName service..."
sc.exe create $serviceName `
    binPath= "`"$exePath`"" `
    start= auto `
    DisplayName= "Prometheus IPMI Exporter" | Out-Null

sc.exe description $serviceName "Prometheus exporter for IPMI System Event Log (SEL) hardware alerts" | Out-Null

Write-Host "Starting $serviceName service..."
Start-Service -Name $serviceName

Write-Host "Prometheus IPMI Exporter installed and running."
