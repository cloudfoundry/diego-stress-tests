package parser

import (
	"encoding/json"
	"io/ioutil"

	"code.cloudfoundry.org/lager"
)

type App struct {
	Name  string   `json:"app_name"`
	Guid  string   `json:"app_guid"`
	Url   string   `json:"app_url"`
	Start AppStart `json:"start"`
}

type AppStart struct {
	Succeeded bool `json:"succeeded"`
}

type AppFile struct {
	Succeeded bool   `json:"succeeded"`
	Apps      []*App `json:"apps"`
}

func ParseAppFile(logger lager.Logger, appFilePath string) ([]*App, error) {
	logger = logger.Session("parser")
	appFileContents, err := ioutil.ReadFile(appFilePath)
	if err != nil {
		logger.Error("failed-to-read-file", err)
		return nil, err
	}

	appFile := AppFile{}
	err = json.Unmarshal(appFileContents, &appFile)
	if err != nil {
		logger.Error("failed-to-unmarshal", err)
		return nil, err
	}

	startedApplications := []*App{}
	for _, app := range appFile.Apps {
		if app.Start.Succeeded {
			startedApplications = append(startedApplications, app)
		}
	}

	return startedApplications, nil
}
