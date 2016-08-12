package cedar_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var configContent string = `
[
  {
    "manifestPath": "assets/manifests/manifest-light.yml",
    "appCount": 9,
    "appNamePrefix": "light"
  },
  {
    "manifestPath": "assets/manifests/manifest-medium-group.yml",
    "appCount": 3,
    "appNamePrefix": "medium-group"
  }
]`

var tempDir, generatedFile string
var fakeLogger lager.Logger

func TestCedar(t *testing.T) {
	BeforeEach(func() {
		tempDir, err := ioutil.TempDir("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		generatedFile = filepath.Join(tempDir, "sample-config.json")

		err = ioutil.WriteFile(generatedFile, []byte(configContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		fakeLogger = lager.NewLogger("fakelogger")
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "Cedar Suite")
}
