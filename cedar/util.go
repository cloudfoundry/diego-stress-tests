package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
	"golang.org/x/net/context"
)

func setupCFCLI(ctx context.Context) error {
	var err error
	logger := ctx.Value("logger").(lager.Logger)

	_, err = cf(ctx, "api", *cfAPI, fmt.Sprintf("--skip-ssl-validation=%t", *skipSSLValidation))
	if err != nil {
		logger.Error("failed-setting-cf-api", nil, lager.Data{"api": *cfAPI})
		return err
	}

	_, err = cf(ctx, "auth", *adminUser, *adminPassword)
	if err != nil {
		logger.Error("failed-cf-auth", err)
		return err
	}

	_, err = cf(ctx, "create-org", *orgName)
	if err != nil {
		logger.Error("failed-creating-org", err, lager.Data{"org": *orgName})
		return err
	}

	_, err = cf(ctx, "create-space", *spaceName, "-o", *orgName)
	if err != nil {
		logger.Error("failed-creating-space", err, lager.Data{"space": *spaceName})
		return err
	}

	_, err = cf(ctx, "target", "-o", *orgName, "-s", *spaceName)
	if err != nil {
		logger.Error("failed-targeting-space", err, lager.Data{"org": *orgName, "space": *spaceName})
		return err
	}

	_, err = cf(ctx, "create-quota", "runaway", "-m", "99999G", "-s", "10000000", "-r", "10000000", "--allow-paid-service-plans")
	if err != nil {
		logger.Error("failed-creating-quota", err)
		return err
	}

	_, err = cf(ctx, "set-quota", *orgName, "runaway")
	if err != nil {
		logger.Error("failed-setting-quota", err)
		return err
	}

	return nil
}

func cf(ctx context.Context, args ...string) ([]byte, error) {
	// Use logger from context object, or create logger if none provided
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("cf", lager.Data{"args": args})

	cmd := exec.Command("cf", args...)
	c := make(chan error, 1)
	var output []byte = nil

	go func() {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logger.Error("failed-starting-cf-command", err)
			c <- err
		}
		err = cmd.Start()
		if err != nil {
			logger.Error("failed-starting-cf-command", err)
			c <- err
		}

		output, err = ioutil.ReadAll(stdout)
		if err != nil {
			logger.Error("failed-starting-cf-command", err)
			c <- err
		}

		err = cmd.Wait()
		if err != nil {
			logger.Error("failed-running-cf-command", err)
			c <- err
		}
		c <- nil
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-c:
		if err != nil {
			return nil, err
		} else {
			return output, nil
		}
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
