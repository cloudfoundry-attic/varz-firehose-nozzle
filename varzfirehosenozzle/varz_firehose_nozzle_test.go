package varzfirehosenozzle_test

import (
	. "github.com/cloudfoundry-incubator/varz-firehose-nozzle/varzfirehosenozzle"

	"errors"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/loggertesthelper"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("VarzFirehoseNozzle", func() {
	var fakeFirehoseConsumer *FakeFirehoseConsumer
	var fakeVarzEmitter *FakeEmitter
	var logger *gosteno.Logger
	var nozzle *VarzFirehoseNozzle

	BeforeEach(func() {
		fakeFirehoseConsumer = &FakeFirehoseConsumer{readyChan: make(chan struct{})}
		fakeVarzEmitter = &FakeEmitter{readyChan: make(chan struct{})}
		logger = loggertesthelper.Logger()
		nozzle = NewVarzFirehoseNozzle("firehose-a", "auth-token", fakeFirehoseConsumer, fakeVarzEmitter, logger)
	})

	It("adds metrics to the VarzEmitter when it receives metrics from the firehose", func() {
		go nozzle.Run()

		<-fakeFirehoseConsumer.readyChan
		envelope := &events.Envelope{}
		fakeFirehoseConsumer.messageChan <- envelope

		<-fakeVarzEmitter.readyChan
		Eventually(func() []*events.Envelope { return fakeVarzEmitter.metrics }).Should(HaveLen(1))
	})

	It("shuts down when the message chan is closed", func() {
		exitChan := make(chan struct{})
		go func() {
			nozzle.Run()
			close(exitChan)
		}()

		<-fakeFirehoseConsumer.readyChan
		close(fakeFirehoseConsumer.messageChan)

		Eventually(exitChan).Should(BeClosed())
		Expect(fakeFirehoseConsumer.closed).To(BeTrue())
	})

	It("shuts down when the error chan is closed", func() {
		exitChan := make(chan struct{})
		go func() {
			nozzle.Run()
			close(exitChan)
		}()

		<-fakeFirehoseConsumer.readyChan
		close(fakeFirehoseConsumer.errorChan)

		Eventually(exitChan).Should(BeClosed())
		Expect(fakeFirehoseConsumer.closed).To(BeTrue())
	})

	It("logs an error when an error occurs", func() {
		go nozzle.Run()

		<-fakeFirehoseConsumer.readyChan
		fakeFirehoseConsumer.errorChan <- errors.New("some error")

		errorMessage := "Error while reading from the firehose: some error"
		Eventually(loggertesthelper.TestLoggerSink.LogContents).Should(ContainSubstring(errorMessage))
	})

})

type FakeEmitter struct {
	metrics   []*events.Envelope
	readyChan chan struct{}
}

func (e *FakeEmitter) AddMetric(envelope *events.Envelope) {
	e.metrics = append(e.metrics, envelope)
	close(e.readyChan)
}

type FakeFirehoseConsumer struct {
	messageChan chan<- *events.Envelope
	errorChan   chan<- error
	closed      bool
	readyChan   chan struct{}
}

func (f *FakeFirehoseConsumer) Firehose(subscriptionId string, authToken string, msgChan chan<- *events.Envelope, errChan chan<- error) {
	f.messageChan = msgChan
	f.errorChan = errChan
	close(f.readyChan)
}

func (f *FakeFirehoseConsumer) Close() error {
	f.closed = true
	return nil
}
