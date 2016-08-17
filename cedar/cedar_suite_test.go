package main_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var configContent string = `
[
  {
    "manifestPath": "manifest-light.yml",
    "appCount": 9,
    "appNamePrefix": "light"
  },
  {
    "manifestPath": "manifest-medium-group.yml",
    "appCount": 3,
    "appNamePrefix": "medium-group"
  }
]`

var tempDir, fakeConfigFile string
var fakeLogger lager.Logger

func TestCedar(t *testing.T) {
	BeforeEach(func() {
		tempDir, err := ioutil.TempDir("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		fakeConfigFile = filepath.Join(tempDir, "sample-config.json")
		err = ioutil.WriteFile(fakeConfigFile, []byte(configContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		fakeLogger = lagertest.NewTestLogger("fakelogger")
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "Cedar Suite")
}
