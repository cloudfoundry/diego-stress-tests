package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/diego-stress-tests/cedar/cli"
	"code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
	"code.cloudfoundry.org/lager"
)

const (
	CF_HOME = "CF_HOME"
)

var (
	cedarOutput = flag.String("o", "output.json", "Cedar output receipt")
)

func main() {
	cflager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger, _ := cflager.New("mulch")
	ctx := context.WithValue(
		context.Background(),
		"logger",
		logger,
	)
	cfClient := cli.NewCfClient(ctx, 1)
	defer cfClient.Cleanup(ctx)

	apps, err := getSpaceSummary(logger, cfClient)
	if err != nil {
		os.Exit(1)
	}
	err = generateReport(logger, apps)
	if err != nil {
		os.Exit(1)
	}
}

type SpaceSummary struct {
	Apps []App
}
type App struct {
	Guid  string
	Name  string
	State string
}

func getSpaceSummary(logger lager.Logger, cfClient cli.CFClient) ([]App, error) {
	logger = logger.Session("space-summary")

	guid, err := getSpaceGuid()
	if err != nil {
		logger.Error("failed-to-get-space-guid", err)
		return nil, err
	}

	spaceSummary, err := cfClient.Cf(logger, context.Background(), 30*time.Second, "curl", "/v2/spaces/"+guid+"/summary")
	if err != nil {
		logger.Error("could-not-get-space-summary", err)
		return nil, err
	}
	var summary SpaceSummary
	decoder := json.NewDecoder(bytes.NewReader(spaceSummary))
	err = decoder.Decode(&summary)
	if err != nil {
		logger.Error("could-not-decode-space-summary", err)
		return nil, err
	}
	return summary.Apps, nil
}

type CLIConfig struct {
	SpaceFields SpaceFields
}

type SpaceFields struct {
	GUID string
}

func getSpaceGuid() (string, error) {
	cfHome := os.Getenv(CF_HOME)
	if cfHome == "" {
		user, err := user.Current()
		if err != nil {
			return "", err
		}
		cfHome = filepath.Join(user.HomeDir, ".cf")
	}
	configPath := filepath.Join(cfHome, "config.json")

	f, err := os.Open(configPath)
	if err != nil {
		return "", err
	}

	var config CLIConfig
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)
	if err != nil {
		return "", err
	}

	return config.SpaceFields.GUID, nil
}

func generateReport(logger lager.Logger, apps []App) error {
	report := seeder.CedarReport{
		true,
		[]seeder.AppStateMetrics{},
	}

	outputFile, err := os.OpenFile(*cedarOutput, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	defer outputFile.Close()
	if err != nil {
		logger.Error("error-opening-file", err)
		return err
	}

	jsonParser := json.NewEncoder(outputFile)
	for _, app := range apps {
		appStateMetric := &seeder.AppStateMetrics{
			AppName: &app.Name,
			AppGuid: &app.Guid,
			PushState: &seeder.State{
				Succeeded: true,
			},
			StartState: &seeder.State{
				Succeeded: app.State == "STARTED",
			},
		}
		report.Apps = append(report.Apps, *appStateMetric)
	}
	jsonParser.Encode(report)

	return nil
}
