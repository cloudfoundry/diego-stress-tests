package cedar

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

	appsToPush  []CfApp
	AppsToStart []CfApp
	Report      map[string]*MetricsReport
}

func NewPusher(config Config, apps []CfApp) Pusher {
	p := Pusher{
		errChan:    make(chan error, config.maxFailures),
		Report:     make(map[string]*MetricsReport),
		config:     config,
		appsToPush: apps,
	}
	return p
}

func (p *Pusher) PushApps(ctx context.Context, cancel context.CancelFunc) {
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("pushing-apps", lager.Data{"max-allowed-failures": p.config.maxFailures})
	logger.Info("started")
	defer logger.Info("complete")

	var seedMutex sync.Mutex

	wg := sync.WaitGroup{}
	rateLimiter := make(chan struct{}, p.config.MaxInFlight)

	for i := 0; i < len(p.appsToPush); i++ {
		seedApp := p.appsToPush[i]
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
			case <-ctx.Done():
				return
			default:
				succeeded = true
				startTime = time.Now()
				err = seedApp.Push(ctx, p.config.AppPayload, p.config.TimeoutDuration())
				endTime = time.Now()
			}

			if err != nil {
				logger.Error("failed-pushing-app", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1})
				succeeded = false
				select {
				case p.errChan <- err:
				default:
					logger.Error("failure-tolerance-reached", nil)
					cancel()
				}
			} else {
				guid, err = seedApp.Guid(ctx, p.config.TimeoutDuration())
				if err != nil {
					logger.Error("failed-getting-app-guid", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1})
				}

				seedMutex.Lock()
				p.AppsToStart = append(p.AppsToStart, seedApp)
				seedMutex.Unlock()
			}

			name := seedApp.AppName()
			p.Report[name] = &MetricsReport{
				AppName:     &name,
				AppGuid:     &guid,
				PushReport:  &Report{},
				StartReport: &Report{},
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
	logger = logger.Session("starting-apps", lager.Data{"max-allowed-failures": p.config.maxFailures})
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
				return
			default:
				succeeded = true
				startTime = time.Now()
				err = appToStart.Start(ctx, p.config.TimeoutDuration())
				endTime = time.Now()
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
	report := MetricsOutput{
		Tolerance: p.config.maxFailures,
		Failed:    len(p.errChan) + 1,
	}
	metricsFile, err := os.OpenFile(p.config.OutputFile, os.O_RDWR|os.O_CREATE, 0644)
	defer metricsFile.Close()

	if err != nil {
		logger.Error("error-opening-metrics-output-file", err)
		os.Exit(1)
	}

	jsonParser := json.NewEncoder(metricsFile)
	for _, value := range p.Report {
		report.Apps = append(report.Apps, *value)
	}
	jsonParser.Encode(report)
}

func (p *Pusher) updateReport(reportType, name string, succeeded bool, startTime, endTime time.Time) {
	var report *Report
	switch reportType {
	case Push:
		report = p.Report[name].PushReport
	case Start:
		report = p.Report[name].StartReport
	}
	start := strconv.FormatInt(startTime.UnixNano(), 10)
	end := strconv.FormatInt(endTime.UnixNano(), 10)
	duration := strconv.FormatInt(endTime.UnixNano()-startTime.UnixNano(), 10)

	report.Succeeded = succeeded
	report.StartTime = &start
	report.EndTime = &end
	report.Duration = &duration
}
