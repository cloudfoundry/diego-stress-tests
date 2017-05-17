package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"code.cloudfoundry.org/bbs"
	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/diego-stress-tests/drd/diagnosis"
	"code.cloudfoundry.org/diego-stress-tests/drd/parser"
	"code.cloudfoundry.org/lager"
)

const (
	clientSessionCacheSize int = 0
	maxIdleConnsPerHost    int = 0
	GUIDByteLength         int = 36
)

var (
	cedarInput        = flag.String("cedarInput", "", "cedar input file")
	bbsSkipCertVerify = flag.Bool("bbsSkipCertVerify", false, "when set to true, skips all SSL/TLS certificate verification")
	bbsURL            = flag.String("bbsURL", "", "URL of BBS server to target")
	bbsCertFile       = flag.String("bbsCertFile", "", "path to the TLS client certificate to use during mutual-auth TLS")
	bbsKeyFile        = flag.String("bbsKeyFile", "", "path to the TLS client private key file to use during mutual-auth TLS")
	bbsCACertFile     = flag.String("bbsCACertFile", "", "path the Certificate Authority (CA) file to use when verifying TLS keypairs")

	bbsClient bbs.Client
	err       error
)

func main() {
	cflager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger := lager.NewLogger("drd")

	if !strings.HasPrefix(*bbsURL, "https") {
		bbsClient = bbs.NewClient(*bbsURL)
	} else {
		if *bbsSkipCertVerify {
			bbsClient, err = bbs.NewSecureSkipVerifyClient(
				*bbsURL,
				*bbsCertFile,
				*bbsKeyFile,
				clientSessionCacheSize,
				maxIdleConnsPerHost,
			)
		} else {
			bbsClient, err = bbs.NewSecureClient(
				*bbsURL,
				*bbsCACertFile,
				*bbsCertFile,
				*bbsKeyFile,
				clientSessionCacheSize,
				maxIdleConnsPerHost,
			)
		}
	}
	if err != nil {
		panic(err)
	}

	desiredLRPFilter := models.DesiredLRPFilter{Domain: ""}
	desiredLRPs, err := bbsClient.DesiredLRPs(logger, desiredLRPFilter)
	if err != nil {
		panic(err)
	}

	desiredLRPSet := constructDesiredLRPsMap(desiredLRPs)

	apps, err := parser.ParseAppFile(logger, *cedarInput)
	if err != nil {
		panic(err)
	}

	summary := aggregateSummary(logger, desiredLRPSet, apps)
	summaryBytes, err := json.Marshal(summary)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(summaryBytes))

	eventSource, err := bbsClient.SubscribeToEvents(logger)
	if err != nil {
		panic(err)
	}

	for {
		event, err := eventSource.Next()
		if err != nil {
			panic(err)
		}
		var instanceGuid, state, crashedReason string
		switch e := event.(type) {
		case *models.ActualLRPCreatedEvent:
			instanceGuid = e.ActualLrpGroup.Instance.GetInstanceGuid()
			state = e.ActualLrpGroup.Instance.GetState()
		case *models.ActualLRPRemovedEvent:
			instanceGuid = e.ActualLrpGroup.Instance.GetInstanceGuid()
			state = e.ActualLrpGroup.Instance.GetState()
		case *models.ActualLRPChangedEvent:
			instanceGuid = e.After.Instance.GetInstanceGuid()
			state = e.After.Instance.GetState()
		case *models.ActualLRPCrashedEvent:
			instanceGuid = e.GetInstanceGuid()
			state = models.ActualLRPStateCrashed
			crashedReason = e.GetCrashReason()
		default:
			continue
		}

		if summary.Update(instanceGuid, state, crashedReason) {
			summaryBytes, err := json.Marshal(summary)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(summaryBytes))
		}
	}
}

func aggregateSummary(logger lager.Logger, desiredLRPSet map[string]*models.DesiredLRP, apps []*parser.App) diagnosis.Summary {
	aggregateSummary := diagnosis.Summary{
		TrackedInstances:   []*diagnosis.InstanceInfo{},
		UntrackedInstances: []*diagnosis.InstanceInfo{},
	}
	for _, app := range apps {
		logger := logger.Session("diagnose-app", lager.Data{"name": app.Name, "guid": app.Guid})
		desiredLRP := desiredLRPSet[app.Guid]
		if desiredLRP == nil {
			logger.Error("missing-app-info", fmt.Errorf("app not found"))
			continue
		}

		actualLRPs, err := bbsClient.ActualLRPGroupsByProcessGuid(logger, desiredLRP.ProcessGuid)
		if err != nil {
			logger.Error("failed-reading-actual-lrp", err)
		}
		summary := diagnosis.DiagnoseApp(app, *desiredLRP, actualLRPs)
		aggregateSummary = diagnosis.JoinSummaries(aggregateSummary, summary)
	}
	return aggregateSummary
}

func constructDesiredLRPsMap(desiredLRPs []*models.DesiredLRP) map[string]*models.DesiredLRP {
	desiredSet := map[string]*models.DesiredLRP{}
	for _, d := range desiredLRPs {
		guid := d.GetProcessGuid()[:GUIDByteLength]
		desiredSet[guid] = d
	}
	return desiredSet
}
