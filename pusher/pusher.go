package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"code.cloudfoundry.org/lager"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

type Pusher struct {
	ctx    context.Context
	cancel context.CancelFunc

	directory string
}

func (p Pusher) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := p.ctx.Value("logger").(lager.Logger)
	p.directory = *cfLogsDirectory

	ctx := p.setupOutputFiles(p.ctx, "cf.setup")
	defer ctx.Value("stdout").(io.Closer).Close()
	defer ctx.Value("stderr").(io.Closer).Close()

	err := os.Chdir(*appPath)
	if err != nil {
		logger.Error("failed-changing-into-app-dir", err, lager.Data{"app-path": *appPath})
		return err
	}

	err = setupCFCLI(ctx)
	if err != nil {
		logger.Error("failed-setting-up-cf-cli", err)
		return err
	}

	close(ready)

	go func() {
		defer p.cancel()

		for i := 0; i < *batches; i++ {
			logger := logger.Session("batch", lager.Data{"batch": i + 1})
			logger.Info("starting")
			p.pushes(context.WithValue(ctx, "logger", logger), *batchSize)
			logger.Info("complete")
		}
	}()

	select {
	case <-signals:
		p.cancel()
	case <-p.ctx.Done():
		logger.Info("context-exited")
		err := p.ctx.Err()
		if err != nil && err != context.Canceled {
			logger.Error("context-errored", err)
			return err
		}
	}

	<-signals
	return nil
}

func (p Pusher) setupOutputFiles(ctx context.Context, prefix string) context.Context {
	logger := ctx.Value("logger").(lager.Logger)

	stdout := openFile(logger, path.Join(p.directory, prefix+".stdout.log"))
	ctx = context.WithValue(ctx, "stdout", stdout)

	stderr := openFile(logger, path.Join(p.directory, prefix+".stderr.log"))
	ctx = context.WithValue(ctx, "stderr", stderr)

	ctx = context.WithValue(ctx, "trace", path.Join(p.directory, prefix+".trace.log"))

	return ctx
}

func (p Pusher) pushes(ctx context.Context, count int) {
	logger := ctx.Value("logger").(lager.Logger)

	wg := sync.WaitGroup{}
	for i := 0; i < count; i++ {
		guid := uuid.NewV4().String()
		logger := logger.Session("push", lager.Data{"guid": guid})
		ctx := context.WithValue(ctx, "logger", logger)

		ctx = p.setupOutputFiles(ctx, "cf.push."+guid)
		defer ctx.Value("stdout").(io.Closer).Close()
		defer ctx.Value("stderr").(io.Closer).Close()

		err := generateManifest(*appsDomain, "manifest.yml.template", guid)
		if err != nil {
			logger.Error("failed-generating-app-manifest", err)
			os.Exit(1)
		}

		wg.Add(1)

		go func() {
			logger.Info("starting")
			defer func() {
				logger.Info("complete")
				wg.Done()
			}()

			for tries := *pushRetries; tries > 0; tries-- {
				err := push(ctx, "-f", fmt.Sprintf("manifests/manifest-%s.yml", guid))
				if err != nil {
					logger.Error("failed-pushing-app", nil, lager.Data{"attempt": *pushRetries - tries + 1})
					continue
				}

				return
			}
			logger.Error("giving-up-pushing-app", nil)
			p.cancel()
		}()
	}
	wg.Wait()
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
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "create-org", *orgName)
	if err != nil {
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "create-space", *spaceName, "-o", *orgName)
	if err != nil {
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "target", "-o", *orgName, "-s", *spaceName)
	if err != nil {
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "create-quota", "runaway", "-m", "99999G", "-s", "10000000", "-r", "10000000", "--allow-paid-service-plans")
	if err != nil {
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "set-quota", *orgName, "runaway")
	if err != nil {
		return err
	}

	return nil
}
