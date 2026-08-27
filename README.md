Prometheus IPMI SEL exporter
============================

The exporter scrapes the hardware log of a server and alerts if problems encountered.

```
usage: ipmi_sel_win_exporter.exe [<flags>]

Flags:
  -h, --[no-]help            Show context-sensitive help (also try --help-long
                             and --help-man).
      --ipmiutil.path="C:\\Program Files (x86)\\Sourceforge\\ipmiutil\\ipmiutil.exe"
                             Path to ipmiutil executable.
      --config.path=""       Path to event filter configuration file (optional).
      --web.listen-address=":9101"
                             Address to listen on for metrics.
      --scrape.interval=900  How often to scrape IPMI data in seconds.
      --[no-]version         Show application version.
```
