package main

import (
	"encoding/json"
	"math"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
)

type AppDefinition struct {
	ManifestPath  string `json:"manifestPath"`
	AppNamePrefix string `json:"appNamePrefix"`
	AppCount      int    `json:"appCount"`
}

type Config struct {
	NumBatches       int
	MaxInFlight      int
	MaxPollingErrors int
	Tolerance        float64
	Domain           string
	AppPayload       string
	ConfigFile       string
	OutputFile       string
	Timeout          int

	appTypes           []AppDefinition
	totalAppCount      int
	maxAllowedFailures int
}

func (c *Config) Init(logger lager.Logger) {
	logger = logger.Session("config")

	c.setAppDefinitionTypes(logger)
	c.setAppAndFailureCounts(logger)
}

func (c *Config) AppTypes() []AppDefinition {
	return c.appTypes
}

func (c *Config) MaxAllowedFailures() int {
	return c.maxAllowedFailures
}

func (c *Config) TotalAppCount() int {
	return c.totalAppCount
}

func (c *Config) TimeoutDuration() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

func (c *Config) setAppDefinitionTypes(logger lager.Logger) {
	conf, err := os.Open(c.ConfigFile)
	defer conf.Close()

	if err != nil {
		logger.Error("error-opening-config-file", err)
		os.Exit(1)
	}

	jsonParser := json.NewDecoder(conf)
	if err = jsonParser.Decode(&c.appTypes); err != nil {
		logger.Error("error-parsing-config-file", err)
		os.Exit(1)
	}
	logger.Info("app-types", lager.Data{"size": len(c.appTypes)})
}

func (c *Config) setAppAndFailureCounts(logger lager.Logger) {
	var totalAppCount int
	for _, appDef := range c.appTypes {
		totalAppCount += appDef.AppCount
	}
	c.totalAppCount = c.NumBatches * totalAppCount
	c.maxAllowedFailures = int(math.Floor(c.Tolerance * float64(c.totalAppCount)))
	logger.Info("config-counts", lager.Data{"app-count": c.totalAppCount, "max-failure": c.maxAllowedFailures})
}
