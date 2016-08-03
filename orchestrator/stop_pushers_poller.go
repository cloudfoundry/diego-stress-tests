package main

import (
	"os"
	"time"

	"github.com/hashicorp/consul/api"

	"code.cloudfoundry.org/lager"
)

type StopPushersPoller struct {
	logger             lager.Logger
	expectedNumPushers int
	started            <-chan struct{}
}

func NewStopPushersPoller(logger lager.Logger, expectedNumPushers int, started <-chan struct{}) *StopPushersPoller {
	return &StopPushersPoller{
		logger:             logger.Session("stop-pushers-handler"),
		expectedNumPushers: expectedNumPushers,
		started:            started,
	}
}

func (h StopPushersPoller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	for {
		select {
		case signal := <-signals:
			h.logger.Info("signalled", lager.Data{"signal": signal})
			return nil
		case <-h.started:
			go h.watchForCompletedPushers()
		}
	}
}

func (h StopPushersPoller) watchForCompletedPushers() {
	logger := h.logger.Session("stop-pushers")

	kv := client.KV()
	logger.Info("starting")

	key := "diego-perf/pusher-start"
	ticker := time.NewTicker(time.Duration(5) * time.Second)

	for {
		<-ticker.C
		var completedPushers = fetchCompletedPushers(logger, kv)
		if completedPushers == h.expectedNumPushers {
			logger.Info("deleting-consul-start-key", lager.Data{"key": key})
			_, err = kv.Delete(key, nil)
			if err != nil {
				logger.Error("failed-deleting-start-key", err)
			}

			break
		} else {
			logger.Debug("waiting-pushers-to-be-done", lager.Data{"expectedFinishedPushers": expectedNumPushers, "actualFinishedPushers": completedPushers})
		}
	}
	logger.Info("complete")
}

func fetchCompletedPushers(logger lager.Logger, kv *api.KV) int {
	key := "diego-perf-pushers"
	pairs, _, err := kv.List(key, nil)
	count := 0
	if err != nil {
		logger.Error("failed-getting-pusher-status-keys", err)
	}
	for i := 0; i < len(pairs); i++ {
		if string(pairs[i].Value) == "done" {
			count++
		}
	}
	return count
}
