package main_test

import (
	"github.com/apcera/nats"
	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/registrars/collectorregistrar"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"os/exec"
	"github.com/cloudfoundry/noaa/events"
)

var (
	fakeFirehoseInputChan chan *events.Envelope
)

var _ = Describe("VarzFirehoseNozzle", func() {
	var (
		nozzleSession *gexec.Session
		natsRunner 	  *natsrunner.NATSRunner
	)

	const natsPort = 24484

	BeforeEach(func() {
		var err error
		natsRunner = natsrunner.NewNATSRunner(natsPort)
		natsRunner.Start()

		nozzleCommand := exec.Command(pathToNozzleExecutable, "-config", "fixtures/test_config.json")
		nozzleSession, err = gexec.Start(
			nozzleCommand,
			gexec.NewPrefixedWriter("[o][nozzle] ", GinkgoWriter),
			gexec.NewPrefixedWriter("[e][nozzle] ", GinkgoWriter),
		)
		Expect(err).NotTo(HaveOccurred())

	})

	AfterEach(func() {
		nozzleSession.Kill().Wait()
		natsRunner.Stop()
	})

	It("registers itself with collector over NATS", func() {
		messageChan := make(chan []byte)
		natsClient := natsRunner.MessageBus
		natsClient.Subscribe(collectorregistrar.AnnounceComponentMessageSubject, func(msg *nats.Msg) {
			messageChan <- msg.Data
		})

		Eventually(messageChan).Should(Receive(MatchRegexp(`^\{"type":"MetronAgent","index":42,"host":"[^:]*:1234","uuid":"42-[0-9a-f-]{36}","credentials":\["admin","admin"\]\}$`)))
	})
})
