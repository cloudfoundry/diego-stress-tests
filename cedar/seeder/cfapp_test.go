package seeder_test

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"code.cloudfoundry.org/diego-stress-tests/cedar/cli/fakes"
	. "code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
	"golang.org/x/net/context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

const (
	timeout = 30 * time.Second
)

var _ = Describe("Cfapp", func() {
	var cfApp CfApp
	var fakeClient fakes.FakeCFClient
	var ctx context.Context
	var err error
	var server *ghttp.Server

	BeforeEach(func() {
		ctx, _ = context.WithCancel(
			context.WithValue(context.Background(),
				"logger",
				fakeLogger,
			),
		)
		server = ghttp.NewServer()

		fakeClient = fakes.FakeCFClient{}

		cfApp, err = NewCfApp("test-app", "random-123-domain.com", false, 1, "test-manifest.yml")

		(cfApp.(*CfApplication)).SetUrl(server.URL())

		Expect(cfApp).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Context("When an app is pushed", func() {
		BeforeEach(func() {
			fakeClient.CfReturns([]byte{}, nil)
		})

		It("should push successfully", func() {
			err = cfApp.Push(fakeLogger, ctx, &fakeClient, "random-dir", timeout)
			Expect(fakeLogger).To(gbytes.Say("push.started"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeLogger).To(gbytes.Say("push.completed"))
		})
	})

	Context("When an app is started", func() {
		BeforeEach(func() {
			fakeClient.CfReturns([]byte{}, nil)

			server.RouteToHandler("GET", regexp.MustCompile(".*"), func(resp http.ResponseWriter, req *http.Request) {
				resp.WriteHeader(200)
			})
		})

		It("should log a successful start", func() {
			err = cfApp.Start(fakeLogger, ctx, &fakeClient, false, timeout)

			Expect(err).NotTo(HaveOccurred())

			Expect(fakeLogger).To(gbytes.Say("start.started"))
			Expect(fakeLogger).To(gbytes.Say("start.completed"))
		})

		It("should return error when starting failed", func() {
			fakeClient.CfReturns(nil, errors.New("oops!"))

			err = cfApp.Start(fakeLogger, ctx, &fakeClient, false, timeout)

			Expect(err).To(HaveOccurred())
			Expect(fakeLogger).To(gbytes.Say("start.failed-to-start"))
		})

		It("should curl the app url", func() {
			err = cfApp.Start(fakeLogger, ctx, &fakeClient, false, timeout)

			Expect(err).NotTo(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

	})

	Context("When an app is not routable after starting", func() {
		BeforeEach(func() {
			server.RouteToHandler("GET", regexp.MustCompile(".*"), func(resp http.ResponseWriter, req *http.Request) {
				resp.WriteHeader(404)
			})
		})

		It("should not retry when it's not requested", func() {
			maxFailedCurls := 0
			cfApp, _ = NewCfApp("test-app", "random-123-domain.com", false, maxFailedCurls, "test-manifest.yml")
			(cfApp.(*CfApplication)).SetUrl(server.URL())
			err = cfApp.Start(fakeLogger, ctx, &fakeClient, false, timeout)

			Expect(err).To(HaveOccurred())

			Expect(fakeLogger).NotTo(gbytes.Say("curl.retrying-curl"))
			Expect(fakeLogger).To(gbytes.Say("curl.failed-to-curl"))

			Expect(server.ReceivedRequests()).To(HaveLen(maxFailedCurls + 1))
		})

		It("should retry curl when it's requested", func() {
			maxFailedCurls := 2
			cfApp, _ = NewCfApp("test-app", "random-123-domain.com", false, maxFailedCurls, "test-manifest.yml")
			(cfApp.(*CfApplication)).SetUrl(server.URL())
			err = cfApp.Start(fakeLogger, ctx, &fakeClient, false, timeout)

			Expect(err).To(HaveOccurred())

			Expect(fakeLogger).To(gbytes.Say("curl.retrying-curl"))
			Expect(fakeLogger).To(gbytes.Say("curl.failed-to-curl"))
			Expect(fakeLogger).To(gbytes.Say("start.failed-curling-app"))

			Expect(server.ReceivedRequests()).To(HaveLen(maxFailedCurls + 1))
		})
	})

	Context("When an app guid is requested", func() {
		BeforeEach(func() {
			fakeClient.CfReturns([]byte("fake-guid"), nil)
		})

		It("should return the guid successfully", func() {
			guid, err := cfApp.Guid(fakeLogger, ctx, &fakeClient, timeout)
			Expect(err).NotTo(HaveOccurred())
			Expect(guid).To(Equal("fake-guid"))
			Expect(fakeLogger).To(gbytes.Say("guid.started"))
			Expect(fakeLogger).To(gbytes.Say("guid.completed"))
		})
	})

	Context("When SSL is not required", func() {
		BeforeEach(func() {
			cfApp, err = NewCfApp("test-app", "random-123-domain.com", false, 1, "test-manifest.yml")
		})

		It("should use http in app url", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(cfApp.AppURL()).To(Equal("http://test-app.random-123-domain.com"))
		})
	})

	Context("When SSL is required", func() {
		BeforeEach(func() {
			cfApp, err = NewCfApp("test-app", "random-123-domain.com", true, 1, "test-manifest.yml")
		})

		It("should use https in app url", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(cfApp.AppURL()).To(Equal("https://test-app.random-123-domain.com"))
		})
	})
})
