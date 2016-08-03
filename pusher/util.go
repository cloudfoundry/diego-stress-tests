package main

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"time"

	"code.cloudfoundry.org/lager"
	"golang.org/x/net/context"
)

func cf(ctx context.Context, timeout time.Duration, args ...string) error {
	logger := ctx.Value("logger").(lager.Logger)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stdout := ctx.Value("stdout").(io.Writer)
	stderr := ctx.Value("stderr").(io.Writer)
	trace := ctx.Value("trace").(string)

	logger = logger.Session("cf", lager.Data{"args": args, "timeout": timeout.String()})
	cmd := exec.Command("cf", args...)
	cmd.Env = append(cmd.Env, "CF_TRACE="+trace)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Start()
	if err != nil {
		logger.Error("failed-starting-cf-command", err)
		os.Exit(1)
	}

	errChan := make(chan error)
	go func() {
		errChan <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		err := ctx.Err()
		logger.Error("cf-command-error", err)
		killErr := cmd.Process.Kill()
		if killErr != nil {
			logger.Error("kill-failed", killErr)
		}
		return err
	case err := <-errChan:
		if err != nil {
			logger.Error("failed-running-cf-command", err)
			return err
		}
	}
	return nil
}

func setupCFCLI(ctx context.Context) error {
	var err error
	logger := ctx.Value("logger").(lager.Logger)

	err = cf(ctx, CFDefaultTimeout, "api", *cfAPI, fmt.Sprintf("--skip-ssl-validation=%t", *skipSSLValidation))
	if err != nil {
		logger.Error("failed-setting-cf-api", nil, lager.Data{"api": *cfAPI})
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "auth", *adminUser, *adminPassword)
	if err != nil {
		logger.Error("failed-cf-auth", err)
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "create-org", *orgName)
	if err != nil {
		logger.Error("failed-creating-org", err, lager.Data{"org": *orgName})
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "create-space", *spaceName, "-o", *orgName)
	if err != nil {
		logger.Error("failed-creating-space", err, lager.Data{"space": *spaceName})
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "target", "-o", *orgName, "-s", *spaceName)
	if err != nil {
		logger.Error("failed-targeting-space", err, lager.Data{"org": *orgName, "space": *spaceName})
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "create-quota", "runaway", "-m", "99999G", "-s", "10000000", "-r", "10000000", "--allow-paid-service-plans")
	if err != nil {
		logger.Error("failed-creating-quota", err)
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "set-quota", *orgName, "runaway")
	if err != nil {
		logger.Error("failed-setting-quota", err)
		return err
	}

	return nil
}

func push(ctx context.Context, args ...string) error {
	return cf(ctx, CFPushTimeout, append([]string{"push"}, args...)...)
}

func generateManifest(domain, templatePath, guid string) error {
	templ, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	file, err := os.Create(fmt.Sprintf("manifests/manifest-%s.yml", guid))
	if err != nil {
		return err
	}

	lightNames := []string{}
	for i := 1; i <= 9; i++ {
		lightNames = append(lightNames, fmt.Sprintf("light%d-%s", i, guid))
	}

	mediumNames := []string{}
	for i := 1; i <= 7; i++ {
		mediumNames = append(mediumNames, fmt.Sprintf("medium%d-%s", i, guid))
	}

	heavyNames := []string{}
	for i := 1; i <= 1; i++ {
		heavyNames = append(heavyNames, fmt.Sprintf("heavy%d-%s", i, guid))
	}

	crashingNames := []string{}
	for i := 1; i <= 2; i++ {
		crashingNames = append(crashingNames, fmt.Sprintf("crashing%d-%s", i, guid))
	}

	return templ.Execute(file, map[string]interface{}{
		"domain":          domain,
		"lightGroupName":  fmt.Sprintf("light-group-%s", guid),
		"mediumGroupName": fmt.Sprintf("medium-group-%s", guid),
		"lightNames":      lightNames,
		"mediumNames":     mediumNames,
		"heavyNames":      heavyNames,
		"crashingNames":   crashingNames,
	})
}

func openFile(logger lager.Logger, filename string) *os.File {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		logger.Error("could-not-open-file", err, lager.Data{"file": filename, "cf_logs_directory": *cfLogsDirectory})
		os.Exit(2)
	}
	return file
}
