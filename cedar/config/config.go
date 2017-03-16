package config

import (
	"encoding/json"
	"math"
	"os"
	"time"

	"code.cloudfoundry.org/diego-stress-tests/cedar/cli"
	"code.cloudfoundry.org/lager"
)

type AppDefinition struct {
	ManifestPath  string `json:"manifestPath"`
	AppNamePrefix string `json:"appNamePrefix"`
	AppCount      int    `json:"appCount"`
}

//go:generate counterfeiter -o fakes/fake_config.go . Config
type Config interface {
	NumBatches() int
	MaxInFlight() int
	MaxPollingErrors() int
	// Tolerance() float64
	AppPayload() string
	Prefix() string
	Domain() string
	UseTLS() bool
	SkipVerifyCertificate() bool
	// ConfigFile() string
	OutputFile() string
	Timeout() time.Duration
	TotalAppCount() int
	MaxAllowedFailures() int
	AppTypes() []AppDefinition
}

type config struct {
	numBatches            int
	maxInFlight           int
	maxPollingErrors      int
	tolerance             float64
	domain                string
	useTLS                bool
	skipVerifyCertificate bool
	appPayload            string
	prefix                string
	configFile            string
	outputFile            string
	timeout               time.Duration

	appTypes []AppDefinition
}

func NewConfig(
	logger lager.Logger,
	cfClient cli.CFClient,
	numBatches, maxInFlight, maxPollingErrors int,
	tolerance float64,
	appPayload, prefix, domain, configFile, outputFile string,
	timeout time.Duration,
	useTLS, skipVerifyCertificate bool,
) (Config, error) {
	c := &config{
		numBatches:            numBatches,
		maxInFlight:           maxInFlight,
		maxPollingErrors:      maxPollingErrors,
		tolerance:             tolerance,
		appPayload:            appPayload,
		prefix:                prefix,
		domain:                domain,
		useTLS:                useTLS,
		skipVerifyCertificate: skipVerifyCertificate,
		configFile:            configFile,
		outputFile:            outputFile,
		timeout:               timeout,
	}
	err := c.init(logger, cfClient)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *config) Domain() string {
	return c.domain
}

func (c *config) UseTLS() bool {
	return c.useTLS
}

func (c *config) SkipVerifyCertificate() bool {
	return c.skipVerifyCertificate
}

func (c *config) AppTypes() []AppDefinition {
	return c.appTypes
}

func (c *config) MaxAllowedFailures() int {
	return int(math.Floor(c.tolerance * float64(c.TotalAppCount())))
}

func (c *config) MaxInFlight() int {
	return c.maxInFlight
}

func (c *config) TotalAppCount() int {
	var totalAppCount int
	for _, appDef := range c.appTypes {
		totalAppCount += appDef.AppCount
	}
	totalAppCount *= c.numBatches
	return totalAppCount
}

func (c *config) Timeout() time.Duration {
	return c.timeout
}

func (c *config) MaxPollingErrors() int {
	return c.maxPollingErrors
}

func (c *config) Prefix() string {
	return c.prefix
}

func (c *config) NumBatches() int {
	return c.numBatches
}

func (c *config) OutputFile() string {
	return c.outputFile
}

func (c *config) AppPayload() string {
	return c.appPayload
}

func (c *config) init(logger lager.Logger, cfClient cli.CFClient) error {
	logger = logger.Session("config")

	c.setAppDefinitionTypes(logger)
	if err := c.initializeDomain(logger, cfClient); err != nil {
		return err
	}
	return nil
}

func (c *config) initializeDomain(logger lager.Logger, cfClient cli.CFClient) error {
	if c.domain == "" {
		var err error
		c.domain, err = cli.GetDefaultSharedDomain(logger, cfClient)
		if err != nil {
			return err
		}
	}
	logger.Info("domain-used-for-pushing", lager.Data{"domain": c.domain})
	return nil
}

func (c *config) setAppDefinitionTypes(logger lager.Logger) {
	conf, err := os.Open(c.configFile)
	defer conf.Close()

	if err != nil {
		logger.Error("error-opening-config-file", err)
		panic("error-opening-config-file")
	}

	jsonParser := json.NewDecoder(conf)
	if err = jsonParser.Decode(&c.appTypes); err != nil {
		logger.Error("error-parsing-config-file", err)
		panic("error-parsing-config-file")
	}
	logger.Info("app-types", lager.Data{"size": len(c.appTypes)})
}
