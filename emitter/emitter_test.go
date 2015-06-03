package emitter_test

import (
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/emitter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry/noaa/events"
	"github.com/gogo/protobuf/proto"
)

var _ = Describe("Emitter", func() {
	It("emits the correct varz message for ValueMetrics", func() {
		e := emitter.New()
		metric := 	&events.Envelope{
			Origin: proto.String("fake-origin"),
			EventType: events.Envelope_ValueMetric.Enum(),
			ValueMetric: &events.ValueMetric{
				Name: proto.String("contextName.metricName"),
				Value: proto.Float64(32.0),
				Unit: proto.String("some-unit"),
			},
			Deployment: proto.String("our-deployment"),
			Ip: proto.String("192.168.0.1"),
			Job: proto.String("doppler"),
			Index: proto.String("0"),
		}
		e.AddMetric(metric)

		varzMessage := e.Emit()
		Expect(varzMessage.Contexts).To(HaveLen(1))
		Expect(varzMessage.Contexts[0].Metrics).To(HaveLen(1))

		context := varzMessage.Contexts[0]
		Expect(context.Name).To(Equal("contextName"))
		valueMetric := context.Metrics[0]
		Expect(valueMetric.Name).To(Equal("metricName"))
		Expect(valueMetric.Value).To(BeEquivalentTo(32.0))
		Expect(valueMetric.Tags).To(HaveKeyWithValue("deployment", "our-deployment"))
		Expect(valueMetric.Tags).To(HaveKeyWithValue("ip", "192.168.0.1"))
		Expect(valueMetric.Tags).To(HaveKeyWithValue("job", "doppler"))
		Expect(valueMetric.Tags).To(HaveKeyWithValue("index", "0"))

	})

	It("emits the correct varz message for CounterEvents", func() {
		e := emitter.New()
		metric := 	&events.Envelope{
			Origin: proto.String("fake-origin"),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name: proto.String("contextName.metricName"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(10),
			},
			Deployment: proto.String("our-deployment"),
			Ip: proto.String("192.168.0.1"),
			Job: proto.String("doppler"),
			Index: proto.String("0"),
		}
		e.AddMetric(metric)

		varzMessage := e.Emit()
		Expect(varzMessage.Contexts).To(HaveLen(1))
		Expect(varzMessage.Contexts[0].Metrics).To(HaveLen(1))

		context := varzMessage.Contexts[0]
		Expect(context.Name).To(Equal("contextName"))
		counterEvent := context.Metrics[0]
		Expect(counterEvent.Name).To(Equal("metricName"))
		Expect(counterEvent.Value).To(BeEquivalentTo(10))
		Expect(counterEvent.Tags).To(HaveKeyWithValue("deployment", "our-deployment"))
		Expect(counterEvent.Tags).To(HaveKeyWithValue("ip", "192.168.0.1"))
		Expect(counterEvent.Tags).To(HaveKeyWithValue("job", "doppler"))
		Expect(counterEvent.Tags).To(HaveKeyWithValue("index", "0"))
	})

	It("does not emit a varzmessage if not a CounterEvent or ValueMetric", func() {
		e := emitter.New()
		metric := 	&events.Envelope{
			Origin: proto.String("fake-origin"),
			EventType: events.Envelope_LogMessage.Enum(),
			LogMessage: &events.LogMessage{
				Message: []byte("some log message"),
				MessageType: events.LogMessage_OUT.Enum(),
				Timestamp: proto.Int64(1000000000),
			},
			Deployment: proto.String("our-deployment"),
			Ip: proto.String("192.168.0.1"),
			Job: proto.String("doppler"),
			Index: proto.String("0"),
		}
		e.AddMetric(metric)

		varzMessage := e.Emit()
		Expect(varzMessage.Contexts).To(HaveLen(0))
	})

	It("does not emit a duplicate varz message for an update to an existing metric", func() {
		e := emitter.New()
		metric1 := 	&events.Envelope{
			Origin: proto.String("fake-origin"),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name: proto.String("contextName.metricName"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(10),
			},
			Deployment: proto.String("our-deployment"),
			Ip: proto.String("192.168.0.1"),
			Job: proto.String("doppler"),
			Index: proto.String("0"),
		}
		e.AddMetric(metric1)

		varzMessage := e.Emit()
		Expect(varzMessage.Contexts).To(HaveLen(1))
		Expect(varzMessage.Contexts[0].Metrics).To(HaveLen(1))
		context := varzMessage.Contexts[0]
		Expect(context.Name).To(Equal("contextName"))
		counterEvent := context.Metrics[0]
		Expect(counterEvent.Name).To(Equal("metricName"))
		Expect(counterEvent.Value).To(BeEquivalentTo(10))

		metric2 := 	&events.Envelope{
			Origin: proto.String("fake-origin"),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name: proto.String("contextName.metricName"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(150),
			},
			Deployment: proto.String("our-deployment"),
			Ip: proto.String("192.168.0.1"),
			Job: proto.String("doppler"),
			Index: proto.String("0"),
		}
		e.AddMetric(metric2)

		varzMessage = e.Emit()
		Expect(varzMessage.Contexts).To(HaveLen(1))
		Expect(varzMessage.Contexts[0].Metrics).To(HaveLen(1))
		context = varzMessage.Contexts[0]
		Expect(context.Name).To(Equal("contextName"))
		counterEvent = context.Metrics[0]
		Expect(counterEvent.Name).To(Equal("metricName"))
		Expect(counterEvent.Value).To(BeEquivalentTo(150))
	})

	It("emits a varz message with different context names", func() {
		e := emitter.New()
		metric1 := 	&events.Envelope{
			Origin: proto.String("fake-origin"),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name: proto.String("contextName1.metricName"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(10),
			},
			Deployment: proto.String("our-deployment"),
			Ip: proto.String("192.168.0.1"),
			Job: proto.String("doppler"),
			Index: proto.String("0"),
		}
		e.AddMetric(metric1)

		metric2 := 	&events.Envelope{
			Origin: proto.String("fake-origin"),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name: proto.String("contextName2.metricName"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(100),
			},
			Deployment: proto.String("our-deployment"),
			Ip: proto.String("192.168.0.2"),
			Job: proto.String("metron"),
			Index: proto.String("1"),
		}
		e.AddMetric(metric2)

		varzMessage := e.Emit()
		Expect(varzMessage.Contexts).To(HaveLen(2))
		Expect(varzMessage.Contexts[0].Metrics).To(HaveLen(1))
		context1 := varzMessage.Contexts[0]
		Expect(context1.Name).To(Equal("contextName1"))
		counterEvent1 := context1.Metrics[0]
		Expect(counterEvent1.Name).To(Equal("metricName"))
		Expect(counterEvent1.Value).To(BeEquivalentTo(10))
		Expect(counterEvent1.Tags).To(HaveKeyWithValue("ip", "192.168.0.1"))
		Expect(counterEvent1.Tags).To(HaveKeyWithValue("job", "doppler"))
		Expect(counterEvent1.Tags).To(HaveKeyWithValue("index", "0"))


		context2 := varzMessage.Contexts[1]
		Expect(context2.Name).To(Equal("contextName2"))
		counterEvent2 := context2.Metrics[0]
		Expect(counterEvent2.Name).To(Equal("metricName"))
		Expect(counterEvent2.Value).To(BeEquivalentTo(100))
		Expect(counterEvent2.Tags).To(HaveKeyWithValue("ip", "192.168.0.2"))
		Expect(counterEvent2.Tags).To(HaveKeyWithValue("job", "metron"))
		Expect(counterEvent2.Tags).To(HaveKeyWithValue("index", "1"))
	})

})
