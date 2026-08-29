$ErrorActionPreference = 'Stop'

# Chocolatey runs this script from the *currently installed* package before an
# upgrade or uninstall, while the new package's chocolateyInstall.ps1 has not
# run yet. Stopping the service here releases the handle on the executable so
# the incoming version can be copied over it.
$serviceName = 'ipmi_sel_win_exporter'

$svc = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -ne 'Stopped') {
    Write-Host "Stopping $serviceName service before package modification..."
    Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
    $svc.WaitForStatus('Stopped', '00:00:30')
}
