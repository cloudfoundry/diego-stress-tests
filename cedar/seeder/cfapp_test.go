package seeder_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"code.cloudfoundry.org/diego-stress-tests/cedar/cli/fakes"
	. "code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
	"golang.org/x/net/context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

const (
	timeout = 30 * time.Second
)

var _ = Describe("Cfapp", func() {
	var cfApp CfApp
	var fakeClient fakes.FakeCFClient
	var ctx context.Context
	var err error
	var server *httptest.Server

	BeforeEach(func() {
		ctx, _ = context.WithCancel(
			context.WithValue(context.Background(),
				"logger",
				fakeLogger,
			),
		)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			fmt.Fprintln(w, ``)
		}))

		fakeClient = fakes.FakeCFClient{}

		cfApp, err = NewCfApp("test-app", "random-123-domain.com", 1, "test-manifest.yml")
		(cfApp.(*CfApplication)).SetUrl(server.URL)
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
		})

		It("should start successfully", func() {
			err = cfApp.Start(fakeLogger, ctx, &fakeClient, timeout)
			Expect(fakeLogger).To(gbytes.Say("start.started"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeLogger).To(gbytes.Say("start.completed"))
		})
	})

	Context("When an app guid is requested", func() {
		BeforeEach(func() {
			fakeClient.CfReturns([]byte("fake-guid"), nil)
		})

		It("should push successfully", func() {
			guid, err := cfApp.Guid(fakeLogger, ctx, &fakeClient, timeout)
			Expect(err).NotTo(HaveOccurred())
			Expect(guid).To(Equal("fake-guid"))
			Expect(fakeLogger).To(gbytes.Say("guid.started"))
			Expect(fakeLogger).To(gbytes.Say("guid.completed"))
		})
	})
})
