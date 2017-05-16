package client

import (
	"code.cloudfoundry.org/bbs"
	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/lager"
)

func DesiredLRPs(logger lager.Logger, bbsClient bbs.Client, domain string) ([]*models.DesiredLRP, error) {
	logger = logger.Session("desired-lrps")
	desiredLRPFilter := models.DesiredLRPFilter{Domain: domain}
	return bbsClient.DesiredLRPs(logger, desiredLRPFilter)
}

func ActualLRPGroupsForGuid(logger lager.Logger, bbsClient bbs.Client, processGuid string) ([]*models.ActualLRPGroup, error) {
	logger = logger.Session("actual-lrp-groups-for-guid")
	return bbsClient.ActualLRPGroupsByProcessGuid(logger, processGuid)
}
