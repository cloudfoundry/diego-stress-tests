package main

import (
	"encoding/json"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
)

type State struct {
	Succeeded bool    `json:"succeeded"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Duration  *string `json:"duration"`
}

type AppStateMetrics struct {
	AppName    *string `json:"app_name"`
	AppGuid    *string `json:"app_guid"`
	AppUrl     *string `json:"app_url"`
	PushState  *State  `json:"push"`
	StartState *State  `json:"start"`
}

const (
	Push  = "push"
	Start = "start"
)

type Pusher struct {
	errChan chan error
	config  Config

	AppsToPush  []CfApp
	AppsToStart []CfApp
	AppStates   map[string]*AppStateMetrics
}

func NewPusher(config Config, apps []CfApp) Pusher {
	p := Pusher{
		errChan:    make(chan error, config.maxAllowedFailures),
		AppStates:  make(map[string]*AppStateMetrics),
		config:     config,
		AppsToPush: apps,
	}
	return p
}

func (p *Pusher) PushApps(ctx context.Context, cancel context.CancelFunc) {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("pushing-apps", lager.Data{"max-allowed-failures": p.config.maxAllowedFailures})
	logger.Info("started")
	defer logger.Info("complete")

	var seedMutex sync.Mutex

	wg := sync.WaitGroup{}
	rateLimiter := make(chan struct{}, p.config.MaxInFlight)

	for i := 0; i < len(p.AppsToPush); i++ {
		seedApp := p.AppsToPush[i]
		wg.Add(1)
		go func() {
			rateLimiter <- struct{}{}
			defer func() {
				<-rateLimiter
				wg.Done()
			}()

			var err error
			var succeeded bool
			var startTime, endTime time.Time
			var guid *string

			select {
			case <-ctx.Done():
				logger.Info("push-cancelled-before-pushing-app", lager.Data{"AppName": seedApp.AppName()})
				return
			default:
				succeeded = true
				startTime = time.Now()
				err = seedApp.Push(ctx, p.config.AppPayload, p.config.TimeoutDuration())
				endTime = time.Now()
			}

			if err != nil {
				logger.Error("failed-pushing-app", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1, "AppName": seedApp.AppName()})
				succeeded = false
				select {
				case p.errChan <- err:
				default:
					logger.Error("failure-tolerance-reached", nil)
					cancel()
				}
			} else {
				appGuid, err := seedApp.Guid(ctx, p.config.TimeoutDuration())
				if err != nil {
					logger.Error("failed-getting-app-guid", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1})
					guid = nil
				} else {
					guid = &appGuid
				}

				seedMutex.Lock()
				logger.Info("pushed-app-and-retrieved-guid", lager.Data{"AppName": seedApp.AppName()})
				p.AppsToStart = append(p.AppsToStart, seedApp)
				seedMutex.Unlock()
			}

			name := seedApp.AppName()
			url := seedApp.Url()
			p.AppStates[name] = &AppStateMetrics{
				AppName:    &name,
				AppGuid:    guid,
				AppUrl:     &url,
				PushState:  &State{},
				StartState: &State{},
			}
			p.updateReport(Push, name, succeeded, startTime, endTime)

		}()
	}
	wg.Wait()

	logger.Info("done-pushing-apps", lager.Data{"seed-apps": len(p.AppsToStart)})
}

func (p *Pusher) StartApps(ctx context.Context, cancel context.CancelFunc) {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("starting-apps", lager.Data{"max-allowed-failures": p.config.maxAllowedFailures})
	logger.Info("started")
	defer logger.Info("completed")

	wg := sync.WaitGroup{}
	rateLimiter := make(chan struct{}, p.config.MaxInFlight)

	for i := 0; i < len(p.AppsToStart); i++ {
		appToStart := p.AppsToStart[i]

		wg.Add(1)

		go func() {
			rateLimiter <- struct{}{}
			defer func() {
				<-rateLimiter
				wg.Done()
			}()

			var err error
			var succeeded bool
			var startTime, endTime time.Time
			select {
			case <-ctx.Done():
				logger.Info("start-cancelled-before-starting-app", lager.Data{"AppName": appToStart.AppName()})
				return
			default:
				succeeded = true
				startTime = time.Now()
				err = appToStart.Start(ctx, p.config.TimeoutDuration())
				endTime = time.Now()
				logger.Info("started-app", lager.Data{"AppName": appToStart.AppName()})
			}

			if err != nil {
				logger.Error("failed-starting-app", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1})
				succeeded = false
				select {
				case p.errChan <- err:
				default:
					logger.Error("failure-tolerance-reached", nil)
					cancel()
				}
			}
			p.updateReport(Start, appToStart.AppName(), succeeded, startTime, endTime)

		}()
	}
	wg.Wait()
}

func (p *Pusher) GenerateReport(ctx context.Context, cancel context.CancelFunc) {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}

	succeeded := true
	select {
	case <-ctx.Done():
		succeeded = false
	default:
	}

	report := struct {
		Succeeded bool              `json:"succeeded"`
		Apps      []AppStateMetrics `json:"apps"`
	}{
		succeeded,
		[]AppStateMetrics{},
	}

	metricsFile, err := os.OpenFile(p.config.OutputFile, os.O_RDWR|os.O_CREATE, 0644)
	defer metricsFile.Close()

	if err != nil {
		logger.Error("error-opening-metrics-output-file", err)
		os.Exit(1)
	}

	jsonParser := json.NewEncoder(metricsFile)
	for _, value := range p.AppStates {
		report.Apps = append(report.Apps, *value)
	}
	jsonParser.Encode(report)
}

func (p *Pusher) updateReport(reportType, name string, succeeded bool, startTime, endTime time.Time) {
	var report *State
	switch reportType {
	case Push:
		report = p.AppStates[name].PushState
	case Start:
		report = p.AppStates[name].StartState
	}
	start := strconv.FormatInt(startTime.UnixNano(), 10)
	end := strconv.FormatInt(endTime.UnixNano(), 10)
	duration := strconv.FormatInt(endTime.UnixNano()-startTime.UnixNano(), 10)

	report.Succeeded = succeeded
	report.StartTime = &start
	report.EndTime = &end
	report.Duration = &duration
}
