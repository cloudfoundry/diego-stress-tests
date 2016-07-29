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

func (h *PusherUpdatesHandler) PostUpdate(w http.ResponseWriter, req *http.Request) {
	var err error

	decoder := json.NewDecoder(req.Body)
	u := new(Update)
	err = decoder.Decode(&u)
	if err != nil {
		h.logger.Error("failed to decode request", err)
		http.Error(w, err.Error(), 500)
	} else {
		h.logger.Info("finished-pushing", lager.Data{"pusher-id": u.PusherID, "batch": u.Batch})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("update logged"))
	}
}
