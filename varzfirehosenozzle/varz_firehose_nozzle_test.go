package varzfirehosenozzle_test

import (
	. "github.com/cloudfoundry-incubator/varz-firehose-nozzle/varzfirehosenozzle"

	"errors"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/loggertesthelper"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
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
		fakeVarzEmitter = &FakeEmitter{
			metricReadyChan: make(chan struct{}),
			alertReadyChan:  make(chan struct{}),
		}
		logger = loggertesthelper.Logger()
		nozzle = NewVarzFirehoseNozzle("firehose-a", "auth-token", fakeFirehoseConsumer, fakeVarzEmitter, logger)
		loggertesthelper.TestLoggerSink.Clear()
	})

	It("adds metrics to the VarzEmitter when it receives metrics from the firehose", func() {
		go nozzle.Run()

		<-fakeFirehoseConsumer.readyChan
		envelope := &events.Envelope{}
		fakeFirehoseConsumer.messageChan <- envelope

		<-fakeVarzEmitter.metricReadyChan
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
		Expect(fakeVarzEmitter.alerted).To(BeFalse())
	})

	It("alerts about a slowConsumer message", func(done Done) {
		go nozzle.Run()

		<-fakeFirehoseConsumer.readyChan

		slowConsumerMessage := &events.Envelope{
			Origin:    proto.String("doppler"),
			Timestamp: proto.Int64(1000000000),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name:  proto.String("TruncatingBuffer.DroppedMessages"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(1),
			},
			Deployment: proto.String("deployment-name"),
			Job:        proto.String("doppler"),
		}

		fakeFirehoseConsumer.messageChan <- slowConsumerMessage

		<-fakeVarzEmitter.alertReadyChan
		Eventually(func() bool { return fakeVarzEmitter.alerted }).Should(BeTrue())

		Expect(loggertesthelper.TestLoggerSink.LogContents()).To(ContainSubstring("We've intercepted an upstream message which indicates that the nozzle or the TrafficController is not keeping up. Please try scaling up the nozzle."))

		close(done)
	})

	It("alerts about a slowConsumer error", func(done Done) {
		go nozzle.Run()

		<-fakeFirehoseConsumer.readyChan

		closeError := &websocket.CloseError{
			Code: websocket.ClosePolicyViolation,
			Text: "Client did not respond to ping before keep-alive timeout expired.",
		}

		fakeFirehoseConsumer.errorChan <- closeError

		<-fakeVarzEmitter.alertReadyChan
		Eventually(func() bool { return fakeVarzEmitter.alerted }).Should(BeTrue())
		Expect(loggertesthelper.TestLoggerSink.LogContents()).To(ContainSubstring("Disconnected because nozzle couldn't keep up. Please try scaling up the nozzle."))
		close(done)
	})
})

type FakeEmitter struct {
	metrics         []*events.Envelope
	alerted         bool
	metricReadyChan chan struct{}
	alertReadyChan  chan struct{}
}

func (e *FakeEmitter) AddMetric(envelope *events.Envelope) {
	e.metrics = append(e.metrics, envelope)
	close(e.metricReadyChan)
}

func (e *FakeEmitter) AlertSlowConsumerError() {
	e.alerted = true
	close(e.alertReadyChan)
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
