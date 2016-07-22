package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sync"
	"text/template"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"

	"github.com/satori/go.uuid"
	"github.com/tedsuo/ifrit"
)

const (
	CFPushTimeout    = 15 * time.Minute
	CFDefaultTimeout = 30 * time.Second
)

var (
	pushRetries = flag.Int("push-retries", 3, "maximum number of tries for a single push")
	batches     = flag.Int("batches", 10, "number of batches of pushes to run")
	batchSize   = flag.Int("batch-size", 10, "size of each parallel batch of pushes")
	appPath     = flag.String("app-path", "./assets/stress-app", "location of the stress app")

	api               = flag.String("api", "api.bosh-lite.com", "location of the cloud controller api")
	adminUser         = flag.String("admin-user", "admin", "uaa admin user")
	adminPassword     = flag.String("admin-password", "admin", "uaa admin password")
	appsDomain        = flag.String("apps-domain", "bosh-lite.com", "apps domain on cloud controller")
	skipSSLValidation = flag.Bool("skip-ssl-validation", true, "skip ssl validation")

	orgName   = flag.String("org-name", "stress-tests-org", "organization to use for stress tests")
	spaceName = flag.String("space-name", "stress-tests-space", "space to use for stress tests")

	cfLogsDirectory = flag.String("cf-logs-directory", "", "absolute path to directory in which to put cf logs")
)

func main() {
	cflager.AddFlags(flag.CommandLine)

	flag.Parse()

	logger, reconfigurableSink := cflager.New("pusher")
	reconfigurableSink.SetMinLevel(lager.DEBUG)
	logger.Info("starting")
	defer logger.Info("complete")
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "logger", logger)

	runner := Runner{ctx: ctx, cancel: cancel}

	monitor := ifrit.Invoke(runner)
	err := <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}
}

type Runner struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (r Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := r.ctx.Value("logger").(lager.Logger)

	close(ready)

	go func() {
		defer r.cancel()

		stdout := openFile(logger, "cf.setup.stdout.log")
		defer stdout.Close()
		ctx := context.WithValue(r.ctx, "stdout", stdout)

		stderr := openFile(logger, "cf.setup.stderr.log")
		defer stderr.Close()
		ctx = context.WithValue(ctx, "stderr", stderr)

		ctx = context.WithValue(ctx, "trace", path.Join(*cfLogsDirectory, "cf.setup.trace.log"))

		err := os.Chdir(*appPath)
		if err != nil {
			logger.Error("failed-changing-into-app-dir", err, lager.Data{"app-path": *appPath})
			os.Exit(3)
		}

		exitOnErr(cf(ctx, CFDefaultTimeout, "api", *api, fmt.Sprintf("--skip-ssl-validation=%t", *skipSSLValidation)))
		exitOnErr(cf(ctx, CFDefaultTimeout, "auth", *adminUser, *adminPassword))
		exitOnErr(cf(ctx, CFDefaultTimeout, "create-org", *orgName))
		exitOnErr(cf(ctx, CFDefaultTimeout, "create-space", *spaceName, "-o", *orgName))
		exitOnErr(cf(ctx, CFDefaultTimeout, "target", "-o", *orgName, "-s", *spaceName))
		exitOnErr(cf(ctx, CFDefaultTimeout, "create-quota", "runaway", "-m", "99999G", "-s", "10000000", "-r", "10000000", "--allow-paid-service-plans"))
		exitOnErr(cf(ctx, CFDefaultTimeout, "set-quota", *orgName, "runaway"))

		pushes := func(ctx context.Context, count int) {
			logger := ctx.Value("logger").(lager.Logger)

			wg := sync.WaitGroup{}
			for i := 0; i < count; i++ {
				guid := uuid.NewV4().String()
				logger := logger.Session("push", lager.Data{"guid": guid})
				ctx := context.WithValue(ctx, "logger", logger)

				stdout := openFile(logger, "cf.push."+guid+".stdout.log")
				defer stdout.Close()
				ctx = context.WithValue(ctx, "stdout", stdout)

				stderr := openFile(logger, "cf.push."+guid+".stderr.log")
				defer stderr.Close()
				ctx = context.WithValue(ctx, "stderr", stderr)

				ctx = context.WithValue(ctx, "trace", path.Join(*cfLogsDirectory, "cf.push."+guid+".trace.log"))

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
					r.cancel()
				}()
			}
			wg.Wait()
		}

		for i := 0; i < *batches; i++ {
			logger := logger.Session("batch", lager.Data{"batch": i + 1})
			logger.Info("starting")
			pushes(context.WithValue(ctx, "logger", logger), *batchSize)
			logger.Info("complete")
		}
	}()

	select {
	case <-signals:
		r.cancel()
	case <-r.ctx.Done():
		logger.Info("context-exited")
		err := r.ctx.Err()
		if err != nil && err != context.Canceled {
			logger.Error("context-errored", err)
			os.Exit(3)
		}
	}

	return nil
}

func openFile(logger lager.Logger, filename string) *os.File {
	file, err := os.OpenFile(path.Join(*cfLogsDirectory, filename), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		logger.Error("could-not-open-file", err, lager.Data{"file": filename, "cf_logs_directory": *cfLogsDirectory})
		os.Exit(2)
	}
	return file
}

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

func push(ctx context.Context, args ...string) error {
	return cf(ctx, CFPushTimeout, append([]string{"push"}, args...)...)
}

func exitOnErr(err error) {
	if err != nil {
		os.Exit(1)
	}
}

func generateManifest(domain, templatePath, guid string) error {
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	f, err := os.Create(fmt.Sprintf("manifests/manifest-%s.yml", guid))
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

	err = t.Execute(f, map[string]interface{}{
		"domain":          domain,
		"lightGroupName":  fmt.Sprintf("light-group-%s", guid),
		"mediumGroupName": fmt.Sprintf("medium-group-%s", guid),
		"lightNames":      lightNames,
		"mediumNames":     mediumNames,
		"heavyNames":      heavyNames,
		"crashingNames":   crashingNames,
	})
	if err != nil {
		return err
	}
	return nil
}
