Prometheus IPMI SEL exporter
============================

The exporter scrapes the hardware log of a server and alerts if problems encountered.

Configuration
-------------

All options live in `config.yml`, which the exporter reads from its own
directory (`C:\Program Files\prometheus-ipmi-exporter\config.yml` for the
Chocolatey package). To change an option, edit the file and restart the
service — no reinstall is needed:

```powershell
notepad 'C:\Program Files\prometheus-ipmi-exporter\config.yml'
Restart-Service ipmi_sel_win_exporter
```

Every key is optional; an omitted key keeps its built-in default.

| Key | Default | Meaning |
| --- | --- | --- |
| `ipmiutil_path` | `C:\Program Files (x86)\Sourceforge\ipmiutil\ipmiutil.exe` | Path to the ipmiutil executable. |
| `web_listen_address` | `:9101` | Address to listen on for metrics. |
| `scrape_interval` | `900` | How often to scrape IPMI data, in seconds. |
| `log_file` | *(empty)* | Log destination. Empty means `ipmi_sel_win_exporter.log` next to the executable when running as a service, or stderr in console mode. Rotated to `<name>.old` past 10 MB. |
| `ignore` | see [config.yml](config.yml) | SEL units/events dropped on an exact match. |
| `statusclear` | see [config.yml](config.yml) | Statuses treated as "cleared" on a prefix match. |
| `eventclear` | `[]` | Events treated as "cleared" on a prefix match. |

The service fails to start on an invalid configuration and writes the reason to
the log file.

Command line
------------

The service takes no arguments — it is configured entirely by `config.yml`. The
flags below exist for ad-hoc console runs and override the file.

```
usage: ipmi_sel_win_exporter.exe [<flags>]

Flags:
  -h, --[no-]help              Show context-sensitive help (also try --help-long
                               and --help-man).
      --config.file=""         Path to the configuration file (default:
                               config.yml next to the executable).
      --ipmiutil.path=""       Override ipmiutil_path from the configuration
                               file.
      --web.listen-address=""  Override web_listen_address from the
                               configuration file.
      --scrape.interval=0      Override scrape_interval from the configuration
                               file.
      --[no-]version           Show application version.
```

Building
--------

Requires Go 1.25 or newer. The exporter is Windows-only — it links the Windows
service API — so it must be built for `windows/amd64`:

```powershell
go build -o ipmi_sel_win_exporter.exe .
```

From Linux or macOS, cross-compile:

```sh
GOOS=windows GOARCH=amd64 go build -o ipmi_sel_win_exporter.exe .
```

Packaging
---------

The Chocolatey package expects the binary in `choco\tools\`. It is gitignored,
so build it there before packing. `config.yml` is pulled in from the repository
root by the `<files>` manifest in the nuspec, so there is only ever one copy of
it to keep up to date.

```powershell
go build -o choco\tools\ipmi_sel_win_exporter.exe .
choco pack choco\prometheus-ipmi-exporter.nuspec
```

That writes `prometheus-ipmi-exporter.<version>.nupkg` to the current directory.
Bump `<version>` in the nuspec and the `version` constant in `exporter.go`
together — the exporter reports the latter as the `version` label on
`ipmi_alert_details_info`.

To install the package from the local directory as a smoke test:

```powershell
choco install prometheus-ipmi-exporter -s . -y
Get-Service ipmi_sel_win_exporter
Invoke-WebRequest http://localhost:9101/metrics -UseBasicParsing
```

Upgrading in place keeps an existing
`C:\Program Files\prometheus-ipmi-exporter\config.yml`; the packaged defaults
land next to it as `config.yml.example`.

An upgrade stops and removes the service before touching any files: the
installed version's `chocolateyBeforeModify.ps1` stops it, and the new
`chocolateyInstall.ps1` stops and deletes it again before copying. Otherwise
the running service holds an open handle on the executable and the copy fails
with "the process cannot access the file ... because it is being used by
another process". The service is recreated and started at the end of the
install.

[Download](https://github.com/atayts/prometheus-ipmi-sel-exporter/releases/latest/download/ipmi_sel_win_exporter.exe) the latest release.
