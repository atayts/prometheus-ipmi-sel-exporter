package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const version = "1.0.0"

// selEntry represents a parsed IPMI SEL line.
type selEntry struct {
	Unit   string
	Event  string
	Status string
}

var (
	ipmiAlerts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ipmi_alerts_total",
		Help: "Number of IPMI hardware alerts.",
	})
	ipmiStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ipmi_alert_details_info",
		Help: "Details of IPMI alerts.",
	}, []string{"message", "alert_count", "version"})
	ipmiScrapeErrors = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ipmi_scrape_errors_total",
		Help: "Number of errors scraping IPMI data.",
	})
)

func init() {
	prometheus.MustRegister(ipmiAlerts)
	prometheus.MustRegister(ipmiStatus)
	prometheus.MustRegister(ipmiScrapeErrors)
}

func runIpmiutil(ipmiutilPath string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ipmiutilPath, "sel", "-c")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("ERROR: ipmiutil command timed out")
			return nil
		}
		log.Printf("ERROR: ipmiutil failed: %v\nOutput: %s", err, string(out))
		return nil
	}
	return strings.Split(string(out), "\n")
}

// parseSEL parses ipmiutil sel -c output.
// Format: eventId | timestamp | severity | source | unit | event | status
func parseSEL(lines []string) []selEntry {
	var entries []selEntry
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != 7 {
			continue
		}
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		source := parts[3]
		if source != "BMC" {
			continue
		}
		entries = append(entries, selEntry{
			Unit:   parts[4],
			Event:  parts[5],
			Status: parts[6],
		})
	}
	return entries
}

func analyze(entries []selEntry, cfg *Config) []string {
	type unitEvent struct {
		Event  string
		Status string
	}
	last := make(map[string]unitEvent)
	for _, e := range entries {
		last[e.Unit] = unitEvent{Event: e.Event, Status: e.Status}
	}

	var alerts []string
	for unit, ev := range last {
		if matchesAny(unit, cfg.Ignore) {
			continue
		}
		if matchesAny(ev.Event, cfg.Ignore) {
			continue
		}
		if hasPrefix(ev.Status, cfg.StatusClear) {
			continue
		}
		if hasPrefix(ev.Event, cfg.EventClear) {
			continue
		}
		alerts = append(alerts, fmt.Sprintf("%s: %s", unit, ev.Event))
	}
	return alerts
}

func matchesAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if s == p {
			return true
		}
	}
	return false
}

func hasPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func collectMetrics(cfg *Config) {
	lines := runIpmiutil(cfg.IpmiutilPath)
	if lines == nil {
		ipmiScrapeErrors.Inc()
		return
	}

	entries := parseSEL(lines)
	alerts := analyze(entries, cfg)

	ipmiAlerts.Set(float64(len(alerts)))

	// Only emit ipmi_alert_details_info when alerts are present —
	// an absent series means healthy (no alerts).
	ipmiStatus.Reset()
	if len(alerts) > 0 {
		sort.Strings(alerts)
		msg := strings.Join(alerts, "; ")
		log.Printf("WARNING: Found %d alert(s): %s", len(alerts), msg)
		ipmiStatus.WithLabelValues(msg, fmt.Sprintf("%d", len(alerts)), version).Set(1)
	}
	ipmiScrapeErrors.Set(0)
}

// runExporter starts the HTTP server and collection loop, blocking until ctx is cancelled.
func runExporter(ctx context.Context, cfg *Config) {
	log.Printf("Config: listen=%s scrape_interval=%ds ipmiutil=%s ignore=%d patterns statusclear=%d patterns eventclear=%d patterns",
		cfg.ListenAddress, cfg.ScrapeInterval, cfg.IpmiutilPath, len(cfg.Ignore), len(cfg.StatusClear), len(cfg.EventClear))

	log.Println("Performing initial IPMI SEL data collection...")
	collectMetrics(cfg)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Duration(cfg.ScrapeInterval) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				collectMetrics(cfg)
			case <-ctx.Done():
				log.Println("Shutting down collection loop")
				return
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><body><h1>IPMI SEL Windows Exporter</h1><p><a href="/metrics">Metrics</a></p></body></html>`)
	})

	srv := &http.Server{Addr: cfg.ListenAddress, Handler: mux}

	go func() {
		<-ctx.Done()
		log.Println("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("Listening on %s", cfg.ListenAddress)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}

	wg.Wait()
	log.Println("Exporter stopped")
}
