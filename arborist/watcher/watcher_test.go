package watcher_test

import (
	"fmt"
	"net/http"
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
		logger                      *lagertest.TestLogger
		fakeClock                   *fakeclock.FakeClock
		applications                []*parser.App
		duration, interval, timeout time.Duration
		server                      *ghttp.Server
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("arborist")
		fakeClock = fakeclock.NewFakeClock(time.Now())
		duration = 5 * time.Second
		interval = 2 * time.Second
		timeout = 1 * time.Second
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

	Context("when the requests are handled successfully", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.VerifyRequest("GET", "/app-1"),
				ghttp.VerifyRequest("GET", "/app-2"),

				ghttp.VerifyRequest("GET", "/app-1"),
				ghttp.VerifyRequest("GET", "/app-2"),

				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/app-1"),
					ghttp.RespondWith(400, nil, nil),
				),
				ghttp.VerifyRequest("GET", "/app-2"),
			)
		})

		It("should curl the applications every interval and exits after the duration", func() {
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()

				result, err := watcher.CheckRoutability(logger, fakeClock, applications, duration, interval)
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

			server.RouteToHandler("GET", ".*", func(resp http.ResponseWriter, req *http.Request) {
				time.Sleep(2 * time.Second)
				resp.WriteHeader(http.StatusOK)
			})
		})

		It("should curl the applications every interval and exits after the duration", func() {
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()

				result, err := watcher.CheckRoutability(logger, fakeClock, applications, duration, interval)
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
})
