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
	configFile := kingpin.Flag("config.file", "Path to the configuration file (default: config.yml next to the executable).").
		Default("").String()
	configPathAlias := kingpin.Flag("config.path", "Deprecated alias for --config.file.").
		Default("").Hidden().String()

	// The flags below only override what the configuration file says; they
	// exist for ad-hoc console runs. The service is configured by the file.
	ipmiutilPath := kingpin.Flag("ipmiutil.path", "Override ipmiutil_path from the configuration file.").
		Default("").String()
	listenAddr := kingpin.Flag("web.listen-address", "Override web_listen_address from the configuration file.").
		Default("").String()
	scrapeInterval := kingpin.Flag("scrape.interval", "Override scrape_interval from the configuration file.").
		Default("0").Int()

	kingpin.HelpFlag.Short('h')
	kingpin.Version(version)
	kingpin.Parse()

	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to detect service mode: %v", err)
	}

	requestedPath := *configFile
	if requestedPath == "" {
		requestedPath = *configPathAlias
	}
	configPath := configFilePath(requestedPath)

	// Start logging to a file before anything can fail: a service has no
	// console, so configuration errors would otherwise vanish.
	if isService {
		setupLogging(defaultLogPath())
	}

	log.Printf("Starting IPMI SEL Windows Exporter v%s", version)

	cfg, err := loadConfig(configPath, requestedPath != "")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *ipmiutilPath != "" {
		cfg.IpmiutilPath = *ipmiutilPath
	}
	if *listenAddr != "" {
		cfg.ListenAddress = *listenAddr
	}
	if *scrapeInterval != 0 {
		cfg.ScrapeInterval = *scrapeInterval
	}

	if err := cfg.validate(); err != nil {
		log.Fatalf("Invalid configuration (%s): %v", configPath, err)
	}

	if cfg.LogFile != "" {
		setupLogging(cfg.LogFile)
	}

	if isService {
		log.Println("Running as Windows Service")
		runAsService(cfg)
	} else {
		log.Println("Running in console mode (Ctrl+C to stop)")
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		runExporter(ctx, cfg)
	}
}
