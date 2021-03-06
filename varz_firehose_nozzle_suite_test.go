package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/gomega/gexec"
)

func TestVarzFirehoseNozzle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VarzFirehoseNozzle Integration Suite")
}

var pathToNozzleExecutable string

var _ = BeforeSuite(func() {
	var err error
	pathToNozzleExecutable, err = gexec.Build("github.com/cloudfoundry-incubator/varz-firehose-nozzle")
	Expect(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
