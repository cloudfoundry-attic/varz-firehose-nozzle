package emitter_test

import (
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/emitter"

	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/testhelpers"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strconv"
)

var _ = Describe("Emitter", func() {
	It("emits the correct varz message for ValueMetrics", func() {
		e := emitter.New("varz-nozzle")

		job := "doppler"
		index := 0

		metric := &events.Envelope{
			Origin:    proto.String("fake-origin"),
			EventType: events.Envelope_ValueMetric.Enum(),
			ValueMetric: &events.ValueMetric{
				Name:  proto.String("metric.name"),
				Value: proto.Float64(32.0),
				Unit:  proto.String("some-unit"),
			},
			Deployment: proto.String("our-deployment"),
			Ip:         proto.String("192.168.0.1"),
			Job:        proto.String(job),
			Index:      proto.String(strconv.Itoa(index)),
		}
		e.AddMetric(metric)

		varzMessage := e.Emit(job, index)
		Expect(varzMessage.Contexts).To(HaveLen(1))

		context := varzMessage.Contexts[0]
		Expect(context.Metrics).To(HaveLen(1))

		valueMetric := context.Metrics[0]
		Expect(valueMetric.Name).To(Equal("metric.name"))
		Expect(valueMetric.Value).To(BeEquivalentTo(32.0))
		Expect(valueMetric.Tags).To(HaveKeyWithValue("deployment", "our-deployment"))
		Expect(valueMetric.Tags).To(HaveKeyWithValue("ip", "192.168.0.1"))
		Expect(valueMetric.Tags).To(HaveKeyWithValue("job", "doppler"))
		Expect(valueMetric.Tags).To(HaveKeyWithValue("index", "0"))

	})

	It("emits the correct varz message for CounterEvents", func() {
		e := emitter.New("varz-nozzle")

		job := "doppler"
		index := 0

		metric := &events.Envelope{
			Origin:    proto.String("fake-origin"),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name:  proto.String("metric.name"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(10),
			},
			Deployment: proto.String("our-deployment"),
			Ip:         proto.String("192.168.0.1"),
			Job:        proto.String(job),
			Index:      proto.String(strconv.Itoa(index)),
		}
		e.AddMetric(metric)

		varzMessage := e.Emit(job, index)
		Expect(varzMessage.Contexts).To(HaveLen(1))

		context := varzMessage.Contexts[0]
		Expect(context.Metrics).To(HaveLen(1))

		counterEvent := context.Metrics[0]
		Expect(counterEvent.Name).To(Equal("metric.name"))
		Expect(counterEvent.Value).To(BeEquivalentTo(10))
		Expect(counterEvent.Tags).To(HaveKeyWithValue("deployment", "our-deployment"))
		Expect(counterEvent.Tags).To(HaveKeyWithValue("ip", "192.168.0.1"))
		Expect(counterEvent.Tags).To(HaveKeyWithValue("job", "doppler"))
		Expect(counterEvent.Tags).To(HaveKeyWithValue("index", "0"))
	})

	It("does not emit a varzmessage if not a CounterEvent or ValueMetric", func() {
		e := emitter.New("varz-nozzle")

		job := "doppler"
		index := 0

		metric := &events.Envelope{
			Origin:    proto.String("fake-origin"),
			EventType: events.Envelope_LogMessage.Enum(),
			LogMessage: &events.LogMessage{
				Message:     []byte("some log message"),
				MessageType: events.LogMessage_OUT.Enum(),
				Timestamp:   proto.Int64(1000000000),
			},
			Deployment: proto.String("our-deployment"),
			Ip:         proto.String("192.168.0.1"),
			Job:        proto.String(job),
			Index:      proto.String(strconv.Itoa(index)),
		}
		e.AddMetric(metric)

		varzMessage := e.Emit(job, index)
		Expect(varzMessage.Contexts).To(HaveLen(0))
	})

	It("does not emit a duplicate varz message for an update to an existing metric", func() {
		e := emitter.New("varz-nozzle")

		job := "doppler"
		index := 0

		metric1 := &events.Envelope{
			Origin:    proto.String("fake-origin"),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name:  proto.String("metric.name"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(10),
			},
			Deployment: proto.String("our-deployment"),
			Ip:         proto.String("192.168.0.1"),
			Job:        proto.String(job),
			Index:      proto.String(strconv.Itoa(index)),
		}
		e.AddMetric(metric1)

		varzMessage := e.Emit(job, index)
		Expect(varzMessage.Contexts).To(HaveLen(1))

		context := varzMessage.Contexts[0]
		Expect(context.Metrics).To(HaveLen(1))

		counterEvent := context.Metrics[0]
		Expect(counterEvent.Name).To(Equal("metric.name"))
		Expect(counterEvent.Value).To(BeEquivalentTo(10))

		metric2 := &events.Envelope{
			Origin:    proto.String("fake-origin"),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name:  proto.String("metric.name"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(150),
			},
			Deployment: proto.String("our-deployment"),
			Ip:         proto.String("192.168.0.1"),
			Job:        proto.String(job),
			Index:      proto.String(strconv.Itoa(index)),
		}
		e.AddMetric(metric2)

		varzMessage = e.Emit(job, index)
		Expect(varzMessage.Contexts).To(HaveLen(1))

		context = varzMessage.Contexts[0]
		Expect(context.Metrics).To(HaveLen(1))

		counterEvent = context.Metrics[0]
		Expect(counterEvent.Name).To(Equal("metric.name"))
		Expect(counterEvent.Value).To(BeEquivalentTo(150))
	})

	It("emits a varz message with different context names", func() {
		e := emitter.New("varz-nozzle")

		job := "doppler"
		index := 0

		contextName1 := "fake-origin-1"
		metric1 := &events.Envelope{
			Origin:    proto.String(contextName1),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name:  proto.String("metric.name"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(10),
			},
			Deployment: proto.String("our-deployment"),
			Ip:         proto.String("192.168.0.1"),
			Job:        proto.String(job),
			Index:      proto.String(strconv.Itoa(index)),
		}
		e.AddMetric(metric1)

		contextName2 := "fake-origin-2"
		metric2 := &events.Envelope{
			Origin:    proto.String(contextName2),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name:  proto.String("metric.name"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(100),
			},
			Deployment: proto.String("our-deployment"),
			Ip:         proto.String("192.168.0.1"),
			Job:        proto.String(job),
			Index:      proto.String(strconv.Itoa(index)),
		}
		e.AddMetric(metric2)

		varzMessage := e.Emit(job, index)
		Expect(varzMessage.Contexts).To(HaveLen(2))

		context1 := testhelpers.FindContext("fake-origin-1", varzMessage.Contexts)
		Expect(context1).NotTo(BeNil())
		Expect(context1.Metrics).To(HaveLen(1))

		context2 := testhelpers.FindContext("fake-origin-2", varzMessage.Contexts)
		Expect(context2).NotTo(BeNil())
		Expect(context2.Metrics).To(HaveLen(1))

		Expect(context1.Metrics).To(HaveLen(1))
		Expect(context2.Metrics).To(HaveLen(1))

		Expect(context1.Metrics[0]).To(Equal(
			emitter.VarzMetric{
				Name:  "metric.name",
				Value: uint64(10),
				Tags: map[string]interface{}{
					"ip":         "192.168.0.1",
					"job":        job,
					"index":      strconv.Itoa(index),
					"deployment": "our-deployment",
				},
			},
		))

		Expect(context2.Metrics[0]).To(Equal(
			emitter.VarzMetric{
				Name:  "metric.name",
				Value: uint64(100),
				Tags: map[string]interface{}{
					"ip":         "192.168.0.1",
					"job":        job,
					"index":      strconv.Itoa(index),
					"deployment": "our-deployment",
				},
			},
		))
	})

	It("emits a varz message with different metric names", func() {
		e := emitter.New("varz-nozzle")

		contextName := "fake-origin-1"

		job1 := "doppler"
		index1 := 0
		metric1 := &events.Envelope{
			Origin:    proto.String(contextName),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name:  proto.String("metric.name"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(10),
			},
			Deployment: proto.String("our-deployment"),
			Ip:         proto.String("192.168.0.1"),
			Job:        proto.String(job1),
			Index:      proto.String(strconv.Itoa(index1)),
		}
		e.AddMetric(metric1)

		job2 := "metron"
		index2 := 1
		metric2 := &events.Envelope{
			Origin:    proto.String(contextName),
			EventType: events.Envelope_CounterEvent.Enum(),
			CounterEvent: &events.CounterEvent{
				Name:  proto.String("metric.name"),
				Delta: proto.Uint64(1),
				Total: proto.Uint64(100),
			},
			Deployment: proto.String("our-deployment"),
			Ip:         proto.String("192.168.0.1"),
			Job:        proto.String(job2),
			Index:      proto.String(strconv.Itoa(index2)),
		}
		e.AddMetric(metric2)

		varzMessage1 := e.Emit(job1, index1)
		Expect(varzMessage1.Contexts).To(HaveLen(1))
		context1 := varzMessage1.Contexts[0]

		varzMessage2 := e.Emit(job2, index2)
		Expect(varzMessage2.Contexts).To(HaveLen(1))
		context2 := varzMessage2.Contexts[0]

		Expect(context1.Metrics).To(HaveLen(1))
		Expect(context2.Metrics).To(HaveLen(1))

		Expect(context1.Metrics[0]).To(Equal(
			emitter.VarzMetric{
				Name:  "metric.name",
				Value: uint64(10),
				Tags: map[string]interface{}{
					"ip":         "192.168.0.1",
					"job":        job,
					"index":      strconv.Itoa(index),
					"deployment": "our-deployment",
				},
			},
		))

		Expect(context2.Metrics[0]).To(Equal(
			emitter.VarzMetric{
				Name:  "metric.name",
				Value: uint64(100),
				Tags: map[string]interface{}{
					"ip":         "192.168.0.1",
					"job":        job,
					"index":      strconv.Itoa(index),
					"deployment": "our-deployment",
				},
			},
		))
	})
	//
	//	It("emits runtime stats in varz message", func() {
	//		e := emitter.New("varz-nozzle")
	//		varzMessage := e.Emit()
	//		Expect(varzMessage.Name).To(Equal("varz-nozzle"))
	//		Expect(varzMessage.NumCpus).To(BeNumerically(">", 0))
	//		Expect(varzMessage.NumGoRoutines).To(BeNumerically(">", 0))
	//		Expect(varzMessage.MemoryStats.BytesAllocated).To(BeNumerically(">", 0))
	//		Expect(varzMessage.MemoryStats.BytesAllocatedHeap).To(BeNumerically(">", 0))
	//		Expect(varzMessage.MemoryStats.BytesAllocatedStack).To(BeNumerically(">", 0))
	//		Expect(varzMessage.MemoryStats.LastGCPauseTimeNS).To(BeNumerically(">", 0))
	//		Expect(varzMessage.MemoryStats.NumFrees).To(BeNumerically(">", 0))
	//		Expect(varzMessage.MemoryStats.NumMallocs).To(BeNumerically(">", 0))
	//	})
	//
	//	It("emits metrics about slow consumer alerts when the nozzle receives an alert", func() {
	//		e := emitter.New("varz-nozzle")
	//		e.AlertSlowConsumerError()
	//
	//		varzMessage := e.Emit()
	//		Expect(varzMessage.Contexts).To(HaveLen(1))
	//
	//		context := testhelpers.FindContext("varz-nozzle", varzMessage.Contexts)
	//		Expect(context).NotTo(BeNil())
	//
	//		Expect(context.Metrics).To(HaveLen(1))
	//		counterMetric := context.Metrics[0]
	//		Expect(counterMetric.Name).To(Equal("slowConsumerAlert"))
	//		Expect(counterMetric.Value).To(BeEquivalentTo(1))
	//	})
	//
	//	It("emits metrics about slow consumer alerts when the nozzle does not receive an alert", func() {
	//		e := emitter.New("varz-nozzle")
	//
	//		varzMessage := e.Emit()
	//		Expect(varzMessage.Contexts).To(HaveLen(1))
	//
	//		context := testhelpers.FindContext("varz-nozzle", varzMessage.Contexts)
	//		Expect(context).NotTo(BeNil())
	//
	//		Expect(context.Metrics).To(HaveLen(1))
	//		counterMetric := context.Metrics[0]
	//		Expect(counterMetric.Name).To(Equal("slowConsumerAlert"))
	//		Expect(counterMetric.Value).To(BeEquivalentTo(0))
	//	})
	//
	//	It("figures out a way to reset the slow consumer alert", func() {
	//		e := emitter.New("varz-nozzle")
	//		e.AlertSlowConsumerError()
	//
	//		varzMessage := e.Emit()
	//		Expect(varzMessage.Contexts).To(HaveLen(1))
	//
	//		context := testhelpers.FindContext("varz-nozzle", varzMessage.Contexts)
	//		Expect(context).NotTo(BeNil())
	//
	//		Expect(context.Metrics).To(HaveLen(1))
	//		counterMetric := context.Metrics[0]
	//		Expect(counterMetric.Name).To(Equal("slowConsumerAlert"))
	//		Expect(counterMetric.Value).To(BeEquivalentTo(1))
	//
	//		varzMessage = e.Emit()
	//		Expect(varzMessage.Contexts).To(HaveLen(1))
	//
	//		context = testhelpers.FindContext("varz-nozzle", varzMessage.Contexts)
	//		Expect(context).NotTo(BeNil())
	//
	//		Expect(context.Metrics).To(HaveLen(1))
	//		counterMetric = context.Metrics[0]
	//		Expect(counterMetric.Name).To(Equal("slowConsumerAlert"))
	//		Expect(counterMetric.Value).To(BeEquivalentTo(0))
	//	})
})
