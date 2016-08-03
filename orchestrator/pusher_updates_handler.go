package main

import (
	"encoding/json"
	"net/http"

	"code.cloudfoundry.org/lager"
)

type PusherUpdatesHandler struct {
	logger lager.Logger
}

type Update struct {
	PusherID string
	Batch    int
}

func NewPusherUpdatesHandler(logger lager.Logger) *PusherUpdatesHandler {
	return &PusherUpdatesHandler{
		logger: logger.Session("pusher-updates-handler"),
	}
}

func (h *PusherUpdatesHandler) PostUpdate(writer http.ResponseWriter, req *http.Request) {
	var err error

	decoder := json.NewDecoder(req.Body)
	update := new(Update)
	err = decoder.Decode(&update)
	if err != nil {
		h.logger.Error("failed to decode request", err)
		http.Error(writer, err.Error(), 500)
	} else {
		h.logger.Info("finished-pushing", lager.Data{"pusher-id": update.PusherID, "batch": update.Batch})
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("update logged"))
	}
}
