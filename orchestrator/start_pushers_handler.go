package main

import (
	"net/http"

	"github.com/hashicorp/consul/api"

	"code.cloudfoundry.org/lager"
)

type StartPushersHandler struct {
	logger lager.Logger
}

func NewStartPushersHandler(logger lager.Logger) *StartPushersHandler {
	return &StartPushersHandler{
		logger: logger.Session("start-pushers-handler"),
	}
}

func (h *StartPushersHandler) StartPushers(w http.ResponseWriter, req *http.Request) {
	var err error
	logger := h.logger.Session("start-pushers")

	kv := client.KV()
	logger.Info("starting")

	key := "diego-perf/pusher-start"
	logger.Debug("writing-consul-start-key", lager.Data{"key": key, "address": *listenAddress})
	_, err = kv.Put(&api.KVPair{Key: key, Value: []byte(*listenAddress)}, nil)
	if err != nil {
		logger.Error("failed-polling-for-key", err)
		http.Error(w, err.Error(), 500)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("started"))
	}
	logger.Info("complete")
}
