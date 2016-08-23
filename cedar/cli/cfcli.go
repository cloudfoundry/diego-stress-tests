package cli

import (
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
	"golang.org/x/net/context"
)

type CFClient interface {
	Cf(ctx context.Context, args ...string) ([]byte, error)
	Cleanup(ctx context.Context)
}

type cfClient struct {
	poolSize int
	pool     chan string
	homeDir  string
}

func NewCfClient(ctx context.Context, poolSize int) CFClient {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("cf")

	user, err := user.Current()
	if err != nil {
		logger.Error("get-home-dir-failed", err)
	}
	homeDir := user.HomeDir

	if _, err = os.Stat(filepath.Join(homeDir, ".cf")); os.IsNotExist(err) {
		logger.Error("cf-dir-unavailable", err)
		os.Exit(1)
	}

	pool := make(chan string, poolSize)
	for i := 0; i < poolSize; i++ {
		cfDir, err := ioutil.TempDir("", "cfhome")
		if err != nil {
			logger.Error("init-temp-cf-dir-failed", err)
		}

		cmd := exec.Command("cp", "-r", filepath.Join(homeDir, ".cf"), filepath.Join(cfDir, ".cf"))
		err = cmd.Run()
		if err != nil {
			logger.Error("copy-cf-config-failed", err)
		}
		pool <- cfDir
	}

	return &cfClient{
		homeDir:  homeDir,
		pool:     pool,
		poolSize: poolSize,
	}
}

func (cfcli *cfClient) Cf(ctx context.Context, args ...string) ([]byte, error) {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("cf", lager.Data{"args": args})

	cfDir := <-cfcli.pool
	os.Setenv("CF_HOME", filepath.Join(cfDir, ".cf"))
	defer func() { cfcli.pool <- cfDir }()

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

func (cfcli *cfClient) Cleanup(ctx context.Context) {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("cf-cleanup")
	logger.Info("started", lager.Data{"tmp-dir-size": len(cfcli.pool)})
	defer logger.Info("completed")

	for tmpDir := range cfcli.pool {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			logger.Error("failed-to-remove-tmpdir", err, lager.Data{"dir": tmpDir})
		}
		if len(cfcli.pool) == 0 {
			close(cfcli.pool)
		}
	}
}
