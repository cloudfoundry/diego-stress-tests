package watcher_test

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/diego-stress-tests/arborist/parser"
	"code.cloudfoundry.org/diego-stress-tests/arborist/watcher"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Watcher", func() {
	var (
		logger             *lagertest.TestLogger
		fakeClock          *fakeclock.FakeClock
		applications       []*parser.App
		duration, interval time.Duration
		server             *ghttp.Server
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("arborist")
		fakeClock = fakeclock.NewFakeClock(time.Now())
		duration = 5 * time.Second
		interval = 2 * time.Second
		server = ghttp.NewServer()

		applications = []*parser.App{
			{
				Name: "app-1",
				Guid: "app-1-guid",
				Url:  fmt.Sprintf("%s/app-1", server.URL()),
			},
			{
				Name: "app-2",
				Guid: "app-2-guid",
				Url:  fmt.Sprintf("%s/app-2", server.URL()),
			},
			{
				Name: "app-3",
				Guid: "app-3-guid",
				Url:  "foobar",
			},
		}

	})

	AfterEach(func() {
		server.Close()
	})

	Context("when the requests are handled successfully", func() {
		var (
			app1Requests = 0
		)

		BeforeEach(func() {
			server.RouteToHandler("GET", "/app-3", func(resp http.ResponseWriter, req *http.Request) {
				resp.WriteHeader(400)
			})

			server.RouteToHandler("GET", "/app-1", func(resp http.ResponseWriter, req *http.Request) {
				app1Requests++
				if app1Requests == 3 {
					resp.WriteHeader(400)
					return
				}
				resp.WriteHeader(http.StatusOK)
			})

			server.RouteToHandler("GET", "/app-2", func(resp http.ResponseWriter, req *http.Request) {
				resp.WriteHeader(200)
			})
		})

		It("should curl the applications every interval and exits after the duration", func() {
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()

				result, err := watcher.CheckRoutability(logger, fakeClock, applications, duration, interval, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeEquivalentTo(map[string]watcher.Result{
					"app-1-guid": watcher.Result{
						Guid:               "app-1-guid",
						Name:               "app-1",
						TotalRequests:      3,
						SuccessfulRequests: 2,
						FailedRequests:     1,
					},
					"app-2-guid": watcher.Result{
						Guid:               "app-2-guid",
						Name:               "app-2",
						TotalRequests:      3,
						SuccessfulRequests: 3,
						FailedRequests:     0,
					},
					"app-3-guid": watcher.Result{
						Guid:               "app-3-guid",
						Name:               "app-3",
						TotalRequests:      3,
						SuccessfulRequests: 0,
						FailedRequests:     3,
					},
				}))
				close(done)
			}()
			// assertions on curls
			Eventually(server.ReceivedRequests).Should(HaveLen(2))

			fakeClock.WaitForWatcherAndIncrement(2 * time.Second)
			Eventually(server.ReceivedRequests).Should(HaveLen(4))

			fakeClock.WaitForWatcherAndIncrement(2 * time.Second)
			Eventually(server.ReceivedRequests).Should(HaveLen(6))

			fakeClock.WaitForWatcherAndIncrement(1 * time.Second)
			Eventually(done).Should(BeClosed())
		})
	})

	Context("when the requests timeout", func() {
		BeforeEach(func() {
			duration = 3 * time.Second

			applications = []*parser.App{
				{
					Name: "app-1",
					Guid: "app-1-guid",
					Url:  fmt.Sprintf("%s/app-1", server.URL()),
				},
			}

			server.RouteToHandler("GET", regexp.MustCompile(".*"), func(resp http.ResponseWriter, req *http.Request) {
				time.Sleep(2 * time.Second)
				resp.WriteHeader(http.StatusOK)
			})
		})

		It("should curl the applications every interval and exits after the duration", func() {
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()

				result, err := watcher.CheckRoutability(logger, fakeClock, applications, duration, interval, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeEquivalentTo(map[string]watcher.Result{
					"app-1-guid": watcher.Result{
						Guid:               "app-1-guid",
						Name:               "app-1",
						TotalRequests:      2,
						SuccessfulRequests: 0,
						FailedRequests:     2,
					},
				}))
				close(done)
			}()

			// assertions on curls
			Eventually(func() int {
				return len(server.ReceivedRequests())
			}, 2*time.Second).Should(Equal(1))

			fakeClock.WaitForWatcherAndIncrement(2 * time.Second)

			Eventually(func() int {
				return len(server.ReceivedRequests())
			}, 2*time.Second).Should(Equal(2))

			fakeClock.WaitForWatcherAndIncrement(1 * time.Second)

			Eventually(done, 2*time.Second).Should(BeClosed())
		})
	})

	Context("curling applications", func() {
		// this test makes sure watcher curl apps concurrently, by sleeping in the
		// handler for 0.5 second and making sure we hit all 3 apps withing a
		// second

		BeforeEach(func() {
			duration = 0               // only check routability once
			interval = 2 * time.Second // hack to set timeout to 1 second

			applications = make([]*parser.App, 0)

			for i := 0; i < 3; i++ {
				applications = append(applications, &parser.App{
					Name: fmt.Sprintf("app-%d", i),
					Guid: fmt.Sprintf("app-%d-guid", i),
					Url:  fmt.Sprintf("%s/app", server.URL()),
				})
			}

			server.RouteToHandler("GET", regexp.MustCompile(".*"), func(resp http.ResponseWriter, req *http.Request) {
				time.Sleep(500 * time.Millisecond)
				resp.WriteHeader(http.StatusOK)
			})
		})

		It("should happen concurently", func() {
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()

				_, err := watcher.CheckRoutability(logger, fakeClock, applications, duration, interval, false)
				Expect(err).NotTo(HaveOccurred())
				close(done)
			}()

			// assertions on curls
			Eventually(func() int {
				return len(server.ReceivedRequests())
			}, time.Second).Should(Equal(3))

			Eventually(done).Should(BeClosed())
		})
	})
})
