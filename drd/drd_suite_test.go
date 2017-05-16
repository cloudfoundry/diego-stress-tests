package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDrd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Drd Suite")
}
