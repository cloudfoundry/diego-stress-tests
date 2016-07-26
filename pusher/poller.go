package main

import (
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
)

type Poller struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (p Poller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := p.ctx.Value("logger").(lager.Logger)
	logger = logger.Session("poller")

	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		logger.Error("failed-building-consul-client", err)
		return err
	}

	kv := client.KV()
	ticker := time.NewTicker(time.Second)

	logger.Info("starting")

	for {
		<-ticker.C

		key := "diego-perf/pusher-start"
		logger.Debug("polling-for-key", lager.Data{"key": key})
		pair, _, err := kv.Get(key, nil)
		if err != nil {
			logger.Error("failed-polling-for-key", err)
			return err
		}

		if pair != nil {
			logger.Info("found-key", lager.Data{"key": key, "value": pair})
			close(ready)
			break
		}
	}

	logger.Info("complete")

	<-signals
	return nil
}
