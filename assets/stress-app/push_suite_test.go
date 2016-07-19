package main_test

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	CFPushTimeout    = 10 * time.Minute
	CFLogsTimeout    = 30 * time.Second
	CFCurlTimeout    = 30 * time.Second
	CurlRetryTimeout = 30 * time.Second
	CurlTimeout      = 10 * time.Second
)

var (
	context helpers.SuiteContext
	config  helpers.Config
)

func TestPush(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Push Suite")
}

var _ = BeforeSuite(func() {
	config = helpers.LoadConfig()
	// context = helpers.NewContext(config)
	// environment := helpers.NewEnvironment(context)

	// environment.Setup()
	// context.SetRunawayQuota()
})
