package main

import (
	"fmt"
	"os"
	"sync"

	"code.cloudfoundry.org/lager"
)

type Pusher struct {
	errChan chan error
	config  Config

	apps []*cfApp
}

func NewPusher(config Config) Pusher {
	return Pusher{
		errChan: make(chan error, config.maxFailures),
		config:  config,
	}
}

func (p *Pusher) PushApps(logger lager.Logger) {
	logger = logger.Session("pushing-apps", lager.Data{"max-allowed-failures": p.config.maxFailures})
	logger.Info("started")
	defer logger.Info("complete")

	var seedMutex sync.Mutex

	wg := sync.WaitGroup{}
	rateLimiter := make(chan struct{}, p.config.maxInFlight)

	for i := 0; i < p.config.numBatches; i++ {
		for _, appDef := range p.config.appTypes {
			for j := 0; j < appDef.AppCount; j++ {
				seedApp := newCfApp(logger, fmt.Sprintf("%s-batch-%d-%d", appDef.AppNamePrefix, i, j), p.config.domain, p.config.maxPollingErrors, appDef.ManifestPath)

				wg.Add(1)

				go func() {
					rateLimiter <- struct{}{}
					defer func() {
						<-rateLimiter
						wg.Done()
					}()

					err := seedApp.Push(logger, p.config.appPayload)

					if err != nil {
						logger.Error("failed-pushing-app", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1})
						select {
						case p.errChan <- err:
						default:
							logger.Error("failure-tolerance-reached", nil)
							os.Exit(1)
						}
					} else {
						seedMutex.Lock()
						p.apps = append(p.apps, seedApp)
						seedMutex.Unlock()
					}
				}()
			}
		}
	}
	wg.Wait()

	logger.Info("done-pushing-apps", lager.Data{"seed-apps": len(p.apps)})
}

func (p *Pusher) StartApps(logger lager.Logger) {
	logger = logger.Session("starting-apps", lager.Data{"max-allowed-failures": p.config.maxFailures})
	logger.Info("started")
	defer logger.Info("completed")

	wg := sync.WaitGroup{}
	rateLimiter := make(chan struct{}, p.config.maxInFlight)

	for i := 0; i < len(p.apps); i++ {
		appToStart := p.apps[i]

		wg.Add(1)

		go func() {
			rateLimiter <- struct{}{}
			defer func() {
				<-rateLimiter
				wg.Done()
			}()

			err := appToStart.Start(logger)

			if err != nil {
				logger.Error("failed-pushing-app", err, lager.Data{"total-incurred-failures": len(p.errChan) + 1})
				select {
				case p.errChan <- err:
				default:
					logger.Error("failure-tolerance-reached", nil)
					os.Exit(1)
				}
			}
		}()
	}
	wg.Wait()
}
