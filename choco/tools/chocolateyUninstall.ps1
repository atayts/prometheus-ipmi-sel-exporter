$ErrorActionPreference = 'Stop'

$serviceName = 'ipmi_sel_win_exporter'
$installDir  = Join-Path $env:ProgramFiles 'prometheus-ipmi-exporter'

# Stop and remove the Windows service.
$svc = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($svc) {
    if ($svc.Status -eq 'Running') {
        Write-Host "Stopping $serviceName service..."
        Stop-Service -Name $serviceName -Force
        $svc.WaitForStatus('Stopped', '00:00:30')
    }
    Write-Host "Removing $serviceName service..."
    sc.exe delete $serviceName | Out-Null
    Start-Sleep -Seconds 2
}

# Remove install directory and all contents.
if (Test-Path $installDir) {
    Write-Host "Removing install directory $installDir..."
    Remove-Item -Path $installDir -Recurse -Force
}

Write-Host "Prometheus IPMI Exporter uninstalled."
