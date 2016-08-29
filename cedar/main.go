package main

import (
	"flag"
	"os"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/diego-stress-tests/cedar/cli"
	"code.cloudfoundry.org/diego-stress-tests/cedar/config"
	"code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
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
	timeout          = flag.Int("timeout", 30, "time allowed for a push or start operation , in seconds")
)

func main() {
	cflager.AddFlags(flag.CommandLine)

	flag.Parse()

	logger, _ := cflager.New("cedar")
	logger.Info("started")
	defer logger.Info("exited")

	config := config.Config{
		NumBatches:       *numBatches,
		MaxInFlight:      *maxInFlight,
		MaxPollingErrors: *maxPollingErrors,
		Tolerance:        *tolerance,
		AppPayload:       *appPayload,
		Prefix:           *prefix,
		Domain:           *domain,
		ConfigFile:       *configFile,
		OutputFile:       *outputFile,
		Timeout:          *timeout,
	}

	config.Init(logger)

	ctx, cancel := context.WithCancel(
		context.WithValue(
			context.Background(),
			"logger",
			logger,
		),
	)

	apps := seeder.NewAppGenerator(config).Apps(ctx)

	cfClient := cli.NewCfClient(ctx, *maxInFlight)
	defer cfClient.Cleanup(ctx)

	if config.Domain == "" {
		var err error
		config.Domain, err = cli.GetDefaultSharedDomain(logger, cfClient)
		if err != nil {
			logger.Error("cannot determine shared domain", err)
			os.Exit(1)
		}
	}

	deployer := seeder.NewDeployer(config, apps, cfClient)
	deployer.PushApps(logger, ctx, cancel)
	deployer.StartApps(ctx, cancel)
	deployer.GenerateReport(ctx, cancel)
}
