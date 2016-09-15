package watcher

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/diego-stress-tests/arborist/parser"
	"code.cloudfoundry.org/lager"
)

type Result struct {
	Guid               string
	Name               string
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
}

func CheckRoutability(logger lager.Logger, clock clock.Clock, applications []*parser.App, duration, interval time.Duration) (map[string]Result, error) {
	logger = logger.Session("watcher")
	results := map[string]Result{}

	durationTimer := clock.NewTimer(duration)
	intervalTicker := clock.NewTicker(interval)

	timeout := interval * 9 / 10

	// initial curling, so we don't have to wait for the intervalTicker to tick
	curlApps(logger, results, applications, timeout)
	for {
		select {
		case <-durationTimer.C():
			// compute result and return
			logger.Info("completed-check-routability")
			return results, nil
		case <-intervalTicker.C():
			logger.Info("initiating-interval-curl")
			curlApps(logger, results, applications, timeout)
		}
	}
}

type curlResult struct {
	app    *parser.App
	passed bool
}

func curlApps(logger lager.Logger, results map[string]Result, applications []*parser.App, timeout time.Duration) {
	resultsCh := make(chan curlResult)

	for _, app := range applications {
		go func(a *parser.App) {
			err := curlApp(logger, a, timeout)
			resultsCh <- curlResult{
				app:    a,
				passed: err == nil,
			}
		}(app)
	}

	for range applications {
		curlResult := <-resultsCh
		app := curlResult.app
		result, ok := results[app.Guid]
		if !ok {
			result = Result{
				Guid: app.Guid,
				Name: app.Name,
			}
		}

		result.TotalRequests++
		if curlResult.passed {
			result.SuccessfulRequests++
		} else {
			result.FailedRequests++
		}

		results[app.Guid] = result
	}
}

func curlApp(logger lager.Logger, app *parser.App, timeout time.Duration) error {
	logger = logger.Session("curl", lager.Data{"url": app.Url, "app-guid": app.Guid})
	logger.Debug("started")
	defer logger.Debug("finished")

	client := http.Client{Timeout: timeout}
	resp, err := client.Get(app.Url)
	if err != nil {
		logger.Error("failed-to-perform-get", err)
		return err
	}

	if resp.StatusCode != 200 {
		err = errors.New(fmt.Sprintf("not a 200, status: %d", resp.StatusCode))
		logger.Error("non-200-get-response", err)
		return err
	}

	return nil
}
