package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/alecthomas/kingpin/v2"
	"golang.org/x/sys/windows/svc"
)

func main() {
	ipmiutilPath := kingpin.Flag("ipmiutil.path", "Path to ipmiutil executable.").
		Default(`C:\Program Files (x86)\Sourceforge\ipmiutil\ipmiutil.exe`).String()
	configPath := kingpin.Flag("config.path", "Path to event filter configuration file (optional).").
		Default("").String()
	listenAddr := kingpin.Flag("web.listen-address", "Address to listen on for metrics.").
		Default(":9101").String()
	scrapeInterval := kingpin.Flag("scrape.interval", "How often to scrape IPMI data in seconds.").
		Default("900").Int()

	kingpin.HelpFlag.Short('h')
	kingpin.Version(version)
	kingpin.Parse()

	log.Printf("Starting IPMI SEL Windows Exporter v%s", version)

	p := exporterParams{
		IpmiutilPath:   *ipmiutilPath,
		ConfigPath:     *configPath,
		ListenAddr:     *listenAddr,
		ScrapeInterval: *scrapeInterval,
	}

	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to detect service mode: %v", err)
	}

	if isService {
		log.Println("Running as Windows Service")
		runAsService(p)
	} else {
		log.Println("Running in console mode (Ctrl+C to stop)")
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		runExporter(ctx, p)
	}
}
