package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	AppRoutePattern = "http://%s.%s"
)

type CfApp struct {
	AppName        string
	AppRoute       url.URL
	AttemptedCurls int
	FailedCurls    int
}

func NewCfApp(appName, rawUrl string) (*CfApp, error) {
	appUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	return &CfApp{
		AppName:  appName,
		AppRoute: *appUrl,
	}, nil
}

func (a *CfApp) Curl(endpoint string) (string, error) {
	endpointUrl := a.AppRoute
	endpointUrl.Path = endpoint

	url := endpointUrl.String()

	statusCode, body, err := curl(url)
	if err != nil {
		return "", err
	}

	a.AttemptedCurls++

	switch {
	case statusCode == 200:
		return string(body), nil

	default:
		err := newCurlErr(url, statusCode, body)
		fmt.Println("FAILED CURL", err.Error())
		a.FailedCurls++
		return "", err
	}
}

func curl(url string) (statusCode int, body string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, "", err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	return resp.StatusCode, string(bytes), nil
}

func newCurlErr(url string, statusCode int, body string) error {
	return fmt.Errorf("Endpoint: %s, Status Code: %d, Body: %s", url, statusCode, body)
}
