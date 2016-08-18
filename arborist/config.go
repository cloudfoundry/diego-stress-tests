package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/lager"
)

type State struct {
	Succeeded bool    `json:"succeeded"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Duration  *string `json:"duration"`
}

type AppStateMetrics struct {
	AppName    *string `json:"app_name"`
	AppGuid    *string `json:"app_guid"`
	AppUrl     *string `json:"app_url"`
	PushState  *State  `json:"push"`
	StartState *State  `json:"start"`
}

type Config struct {
	Succeeded bool              `json:"succeeded"`
	Apps      []AppStateMetrics `json:"apps"`
}

func ReadConfig(logger lager.Logger, fileName string) *Config {
	logger = logger.Session("read-config")

	inputFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		logger.Error("File error", err)
		os.Exit(1)
	}

	var config Config
	json.Unmarshal(inputFile, &config)
	logger.Info("Results", lager.Data{"config": config})
	return &config
}
