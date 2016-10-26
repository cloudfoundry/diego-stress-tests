package main

import (
	"flag"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/diego-stress-tests/cedar/cli"
	"code.cloudfoundry.org/diego-stress-tests/cedar/config"
	"code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
	"code.cloudfoundry.org/lager"
)

var (
	domain = flag.String("domain", "", "app domain")

	numBatches       = flag.Int("n", 1, "number of batches to seed")
	maxInFlight      = flag.Int("k", 1, "max number of cf operations in flight")
	maxPollingErrors = flag.Int("max-polling-errors", 1, "max number of curl failures")
	tolerance        = flag.Float64("tolerance", 1.0, "fractional failure tolerance")
	configFile       = flag.String("config", "config.json", "path to cedar config file")
	outputFile       = flag.String("output", "output.json", "path to cedar metric results file")
	appPayload       = flag.String("payload", "assets/temp-app", "directory containing the stress-app payload to push")
	prefix           = flag.String("prefix", "cedarapp", "the naming prefix for cedar generated apps")
	timeout          = flag.Duration("timeout", 30*time.Second, "time allowed for a push or start operation, golang duration")
)

func main() {
	cflager.AddFlags(flag.CommandLine)

	flag.Parse()

	logger, _ := cflager.New("cedar")
	logger.Info("started")
	defer logger.Info("exited")

	ctx, cancel := context.WithCancel(
		context.WithValue(
			context.Background(),
			"logger",
			logger,
		),
	)
	cfClient := cli.NewCfClient(ctx, *maxInFlight)
	defer cfClient.Cleanup(ctx)

	config, err := config.NewConfig(
		logger,
		cfClient,
		*numBatches,
		*maxInFlight,
		*maxPollingErrors,
		*tolerance,
		*appPayload,
		*prefix,
		*domain,
		*configFile,
		*outputFile,
		*timeout,
	)

	if err != nil {
		logger.Error("failed-to-initialize", err)
		panic("failed-to-initialize")
	}

	apps := generateApps(logger, config)
	deployer := seeder.NewDeployer(config, apps, cfClient)
	deployer.PushApps(logger, ctx, cancel)
	deployer.StartApps(ctx, cancel)
	if succeeded := deployer.GenerateReport(ctx, cancel); !succeeded {
		panic("seeding failed")
	}
}

func generateApps(logger lager.Logger, config config.Config) []seeder.CfApp {
	appsGenerator := seeder.NewAppGenerator(config)
	return appsGenerator.Apps(logger)
}
