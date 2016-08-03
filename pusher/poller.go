package main

import (
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
)

type Poller struct {
	ctx               context.Context
	cancel            context.CancelFunc
	started           chan<- string
	lastModifiedIndex uint64
}

func (p Poller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := p.ctx.Value("logger").(lager.Logger)
	logger = logger.Session("poller")

	close(ready)

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-signals:
			return nil
		case <-ticker.C:
			err := p.checkKey(logger)
			if err != nil {
				logger.Error("failed-checking-key", err)
				return err
			}
		}
	}
}

func (p *Poller) checkKey(logger lager.Logger) error {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		logger.Error("failed-building-consul-client", err)
		return err
	}

	kv := client.KV()

	logger.Debug("starting")

	key := "diego-perf/pusher-start"
	logger.Debug("polling-for-key", lager.Data{"key": key})

	pair, _, err := kv.Get(key, nil)
	if err != nil {
		logger.Error("failed-polling-for-key", err)
		return err
	}

	if pair != nil && pair.ModifyIndex != p.lastModifiedIndex {
		logger.Info("found-key", lager.Data{"key": key, "value": string(pair.Value), "ModifyIndex": pair.ModifyIndex, "LastModifiedIndex": p.lastModifiedIndex})
		p.started <- string(pair.Value)
		p.lastModifiedIndex = pair.ModifyIndex
	}

	logger.Debug("complete")
	return nil
}
