package main

import (
	"context"
	"log"

	"golang.org/x/sys/windows/svc"
)

const serviceName = "ipmi_sel_win_exporter"

type exporterService struct {
	params exporterParams
}

func (s *exporterService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		runExporter(ctx, s.params)
		close(done)
	}()

	changes <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Println("Service stop/shutdown requested")
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				<-done
				return false, 0
			}
		case <-done:
			return false, 1
		}
	}
}

func runAsService(p exporterParams) {
	err := svc.Run(serviceName, &exporterService{params: p})
	if err != nil {
		log.Fatalf("Failed to run service: %v", err)
	}
}
