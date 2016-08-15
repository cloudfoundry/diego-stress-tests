package main

import (
	"encoding/json"
	"math"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
)

type appDefinition struct {
	ManifestPath  string `json:"manifestPath"`
	AppNamePrefix string `json:"appNamePrefix"`
	AppCount      int    `json:"appCount"`
}

type Config struct {
	numBatches       int
	maxInFlight      int
	maxPollingErrors int
	tolerance        float64
	domain           string
	appPayload       string
	configFile       string
	outputFile       string
	timeout          int

	appTypes      []appDefinition
	totalAppCount int
	maxFailures   int
}

func (c *Config) Init(logger lager.Logger) {
	logger = logger.Session("config")
	c.setAppDefinitionTypes(logger)
	c.setAppAndFailureCounts(logger)
}

func (c *Config) Timeout() time.Duration {
	return time.Duration(c.timeout) * time.Second
}

func (c *Config) setAppDefinitionTypes(logger lager.Logger) {
	conf, err := os.Open(c.configFile)
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
	c.totalAppCount = c.numBatches * totalAppCount
	c.maxFailures = int(math.Floor(*tolerance * float64(c.totalAppCount)))
	logger.Info("config-counts", lager.Data{"app-count": c.totalAppCount, "max-failure": c.maxFailures})
}
