package main

import (
	"encoding/json"
	"net/http"

	"code.cloudfoundry.org/lager"
)

type PusherUpdatesHandler struct {
	logger lager.Logger
}

type update struct {
	pusherID string
	batch    int
}

func NewPusherUpdatesHandler(logger lager.Logger) *PusherUpdatesHandler {
	return &PusherUpdatesHandler{
		logger: logger.Session("pusher-updates-handler"),
	}
}

func (h *PusherUpdatesHandler) PostUpdate(w http.ResponseWriter, req *http.Request) {
	var err error

	decoder := json.NewDecoder(req.Body)
	var u update
	err = decoder.Decode(&u)
	if err != nil {
		h.logger.Error("failed to decode request", err)
		http.Error(w, err.Error(), 500)
	} else {
		h.logger.Info("finished-pushing", lager.Data{"pusher-id": u.pusherID, "batch": u.batch})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("update logged"))
	}
}
