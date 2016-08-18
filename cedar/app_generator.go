package main

import (
	"fmt"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
	"golang.org/x/net/context"
)

type appGenerator struct {
	config Config
}

func NewAppGenerator(config Config) appGenerator {
	return appGenerator{
		config: config,
	}
}

func (a appGenerator) Apps(ctx context.Context) []CfApp {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("generating-apps")
	logger.Info("started")
	defer logger.Info("complete")

	apps := []CfApp{}
	for i := 0; i < a.config.NumBatches; i++ {
		for _, appDef := range a.config.appTypes {
			for j := 0; j < appDef.AppCount; j++ {
				name := a.appName(a.config, appDef.AppNamePrefix, i, j)
				logger.Info("generate-app", lager.Data{"appName": name})
				seedApp, err := NewCfApp(name, a.config.Domain, a.config.MaxPollingErrors, appDef.ManifestPath)
				if err != nil {
					logger.Error("failed-generating-app", err)
					continue
				}
				apps = append(apps, seedApp)
			}
		}
	}
	return apps
}

func (a appGenerator) appName(config Config, appName string, batchSeq, appSeq int) string {
	return fmt.Sprintf("%s-%d-%s-%d", config.Prefix, batchSeq, appName, appSeq)
}
