package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/lager"
)

type Report struct {
	Succeeded bool    `json:"succeeded"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Duration  *string `json:"duration"`
}

type MetricsReport struct {
	AppName     *string `json:"app_name"`
	AppGuid     *string `json:"app_guid"`
	PushReport  *Report `json:"push"`
	StartReport *Report `json:"start"`
}

type MetricsOutput struct {
	Tolerance int             `json:"tolerance"`
	Failed    int             `json:"failed"`
	Apps      []MetricsReport `json:"apps"`
}

const (
	Push  = "push"
	Start = "start"
)

type Pusher struct {
	errChan chan error
	config  Config
	ctx     context.Context
	cancel  context.CancelFunc

	apps   []*cfApp
	report map[string]*MetricsReport
}

func NewPusher(config Config) Pusher {
	p := Pusher{
		errChan: make(chan error, config.maxFailures),
		report:  make(map[string]*MetricsReport),
		config:  config,
	}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p
}

func (p *Pusher) PushApps(logger lager.Logger) {
	logger = logger.Session("pushing-apps", lager.Data{"max-allowed-failures": p.config.maxFailures})
	logger.Info("started")
	defer logger.Info("complete")

	var seedMutex sync.Mutex

	wg := sync.WaitGroup{}
	rateLimiter := make(chan struct{}, p.config.maxInFlight)

	for i := 0; i < p.config.numBatches; i++ {
		for _, appDef := range p.config.appTypes {
			for j := 0; j < appDef.AppCount; j++ {
				name := fmt.Sprintf("%s-batch-%d-%d", appDef.AppNamePrefix, i, j)
				seedApp := newCfApp(logger, name, p.config.domain, p.config.maxPollingErrors, appDef.ManifestPath)

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
					var guid string

					select {
					case <-p.ctx.Done():
						return
					default:
						succeeded = true
						startTime = time.Now()
						err = seedApp.Push(logger, p.config.appPayload)
						endTime = time.Now()
					}

					if err != nil {
						logger.Error("failed-pushing-app", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1})
						succeeded = false
						select {
						case p.errChan <- err:
						default:
							logger.Error("failure-tolerance-reached", nil)
							p.cancel()
						}
					} else {
						guid, err = seedApp.Guid(logger)
						if err != nil {
							logger.Error("failed-getting-app-guid", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1})
						}

						seedMutex.Lock()
						p.apps = append(p.apps, seedApp)
						seedMutex.Unlock()
					}

					p.report[name] = &MetricsReport{
						AppName:     &name,
						AppGuid:     &guid,
						PushReport:  &Report{},
						StartReport: &Report{},
					}
					p.updateReport(Push, name, succeeded, startTime, endTime)

				}()
			}
		}
	}
	wg.Wait()

	logger.Info("done-pushing-apps", lager.Data{"seed-apps": len(p.apps)})
}

func (p *Pusher) StartApps(logger lager.Logger) {
	logger = logger.Session("starting-apps", lager.Data{"max-allowed-failures": p.config.maxFailures})
	logger.Info("started")
	defer logger.Info("completed")

	wg := sync.WaitGroup{}
	rateLimiter := make(chan struct{}, p.config.maxInFlight)

	for i := 0; i < len(p.apps); i++ {
		appToStart := p.apps[i]

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
			case <-p.ctx.Done():
				return
			default:
				succeeded = true
				startTime = time.Now()
				err = appToStart.Start(logger)
				endTime = time.Now()
			}

			if err != nil {
				logger.Error("failed-starting-app", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1})
				succeeded = false
				select {
				case p.errChan <- err:
				default:
					logger.Error("failure-tolerance-reached", nil)
					p.cancel()
				}
			}
			p.updateReport(Start, appToStart.appName, succeeded, startTime, endTime)

		}()
	}
	wg.Wait()
}

func (p *Pusher) GenerateReport(logger lager.Logger) {
	report := MetricsOutput{
		Tolerance: p.config.maxFailures,
		Failed:    len(p.errChan) + 1,
	}
	metricsFile, err := os.OpenFile(p.config.outputFile, os.O_RDWR|os.O_CREATE, 0644)
	defer metricsFile.Close()

	if err != nil {
		logger.Error("error-opening-metrics-output-file", err)
		os.Exit(1)
	}

	jsonParser := json.NewEncoder(metricsFile)
	for _, value := range p.report {
		report.Apps = append(report.Apps, *value)
	}
	jsonParser.Encode(report)
}

func (p *Pusher) updateReport(reportType, name string, succeeded bool, startTime, endTime time.Time) {
	var report *Report
	switch reportType {
	case Push:
		report = p.report[name].PushReport
	case Start:
		report = p.report[name].StartReport
	}
	start := strconv.FormatInt(startTime.UnixNano(), 10)
	end := strconv.FormatInt(endTime.UnixNano(), 10)
	duration := strconv.FormatInt(endTime.UnixNano()-startTime.UnixNano(), 10)

	report.Succeeded = succeeded
	report.StartTime = &start
	report.EndTime = &end
	report.Duration = &duration
}
