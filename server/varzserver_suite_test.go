package varzserver_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestVarzserver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Varzserver Suite")
}
