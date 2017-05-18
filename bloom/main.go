package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
)

var (
	concurrency = flag.Int("k", 1, "Number of operations to do concurrently")
	cedarOutput = flag.String("i", "output.json", "Cedar output receipt")
	stopping    = flag.Bool("stop", false, "Whether or not to stop the apps")
)

func main() {
	flag.Parse()

	if len(os.Args) < 1 {
		log.Fatal("startstopper -k <concurrency> -i <output-file> start|stop")
	}

	f, err := os.Open(*cedarOutput)
	if err != nil {
		log.Fatalf("Failed to open cedar output: %s", err.Error())
	}

	var report seeder.CedarReport
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&report)
	if err != nil {
		log.Fatalf("Failed to parse cedar output: %s", err.Error())
	}

	if !report.Succeeded {
		log.Fatal("Cedar did not succeed")
	}

	if *stopping {
		stopApps(report, *concurrency)
	} else {
		startApps(report, *concurrency)
	}
}

func startApps(rep seeder.CedarReport, maxInFlight int) {
	wg := &sync.WaitGroup{}
	rateLimit := make(chan struct{}, maxInFlight)
	for _, application := range rep.Apps {
		app := application
		wg.Add(1)
		go func() {
			rateLimit <- struct{}{}
			defer func() {
				<-rateLimit
				wg.Done()
			}()

			startApp(app)
		}()
	}

	wg.Wait()
}

func startApp(app seeder.AppStateMetrics) {
	if app.StartState.Succeeded {
		log.Printf("Starting Application: %s", *app.AppName)
		err := changeState(*app.AppGuid, "STARTED")
		if err != nil {
			log.Printf("Failed to start application: %s", err.Error())
		} else {
			log.Printf("Succeeded Starting Application: %s", *app.AppName)
		}
	}
}

func stopApps(rep seeder.CedarReport, maxInFlight int) {
	wg := &sync.WaitGroup{}
	rateLimit := make(chan struct{}, maxInFlight)
	for _, application := range rep.Apps {
		app := application
		wg.Add(1)
		go func() {
			rateLimit <- struct{}{}
			defer func() {
				<-rateLimit
				wg.Done()
			}()

			stopApp(app)
		}()
	}

	wg.Wait()
}

func stopApp(app seeder.AppStateMetrics) {
	if app.PushState.Succeeded {
		log.Printf("Stopping Application: %s", *app.AppName)
		err := changeState(*app.AppGuid, "STOPPED")
		if err != nil {
			log.Printf("Failed to stop application, %s, err: %s", *app.AppName, err.Error())
		} else {
			log.Printf("Succeeded Stopping Application: %s", *app.AppName)
		}
	}
}

func changeState(appGuid, state string) error {
	appUrl := fmt.Sprintf("/v2/apps/%s", appGuid)
	cmd := exec.Command("cf", "curl", appUrl, "-X", "PUT", "-d", "{\"state\":\""+state+"\"}")
	return cmd.Run()
}
