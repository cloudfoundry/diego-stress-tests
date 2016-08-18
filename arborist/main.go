package main

import (
	"flag"
	"os"
	"time"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

var (
	rate       = flag.Int("rate", 60, "rate of requests to applications")
	duration   = flag.Int("duration", 10, "duration for tool to make requests in minutes")
	cedarFile  = flag.String("cedar-file", "output.json", "cedar receipt file")
	resultFile = flag.String("result-file", "output.txt", "output of results")
	config     *Config
)

func main() {
	cflager.AddFlags(flag.CommandLine)

	flag.Parse()

	logger, _ := cflager.New("arborist")
	logger.Info("started")
	defer logger.Info("exited")

	if cedarFile != nil {
		config = ReadConfig(logger, *cedarFile)
	}

	for _, app := range config.Apps {
		if app.StartState.Succeeded {
			cfapp, err := NewCfApp(*app.AppName, *app.AppUrl)
			if err != nil {
				logger.Error("unable to create app", err)
				os.Exit(1)
			}
			go performCurl(logger, cfapp, *rate, *duration, clock.NewClock())
		}
	}
}

func performCurl(logger lager.Logger, cfapp *CfApp, rate, duration int, clock clock.Clock) {
	logger = logger.Session("perform-curl", lager.Data{"appName": cfapp.AppName})
	logger.Info("started")
	defer logger.Info("completed")
	completedTimer := clock.NewTimer(time.Minute * time.Duration(duration))

	intervalTime := time.Second * time.Duration(rate)
	sleeper := clock.NewTimer(intervalTime)
	for {
		select {
		case <-completedTimer.C():
			logger.Info("completed-test", lager.Data{"attempted-curls": cfapp.AttemptedCurls, "failed-curls": cfapp.FailedCurls})
			return
		case <-sleeper.C():
			cfapp.Curl("")
			sleeper.Reset(intervalTime)
		}
	}
}
