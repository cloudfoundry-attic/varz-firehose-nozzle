package emitter

import (
	"runtime"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/pivotal-golang/localip"
	"strconv"
)


// Serialization structs
type VarzMetric struct {
	Name  string                 `json:"name"`
	Value interface{}            `json:"value"`
	Tags  map[string]interface{} `json:"tags,omitempty"`
}

func NewVarzMetric(e *events.Envelope) *VarzMetric{
	name, value := getNameAndValue(e)
	if name == "" && value == nil {
		return nil
	}

	tags := computeTags(e)
	return &VarzMetric{
		Name: name,
		Value: value,
		Tags: tags,
	}
}

func getNameAndValue(metric *events.Envelope) (string, interface{}) {
	var name string
	var value interface{}
	switch metric.GetEventType() {
	case events.Envelope_ValueMetric:
		name = metric.GetValueMetric().GetName()
		value = metric.GetValueMetric().GetValue()
	case events.Envelope_CounterEvent:
		name = metric.GetCounterEvent().GetName()
		value = metric.GetCounterEvent().GetTotal()
	default:
		return "", nil
	}
	return name, value
}

func computeTags(metric *events.Envelope) map[string]interface{} {
	tags := make(map[string]interface{})
	tags["deployment"] = metric.GetDeployment()
	tags["ip"] = metric.GetIp()
	tags["job"] = metric.GetJob()
	tags["index"] = metric.GetIndex()
	return tags
}

type VarzContext struct {
	Name    string   `json:"name"`
	Metrics []VarzMetric `json:"metrics"`
}

type VarzMemoryStats struct {
	BytesAllocatedHeap  uint64 `json:"numBytesAllocatedHeap"`
	BytesAllocatedStack uint64 `json:"numBytesAllocatedStack"`
	BytesAllocated      uint64 `json:"numBytesAllocated"`
	NumMallocs          uint64 `json:"numMallocs"`
	NumFrees            uint64 `json:"numFrees"`
	LastGCPauseTimeNS   uint64 `json:"lastGCPauseTimeNS"`
}

type VarzMessage struct {
	Name          string            `json:"name"`
	NumCpus       int               `json:"numCPUS"`
	NumGoRoutines int               `json:"numGoRoutines"`
	MemoryStats   VarzMemoryStats   `json:"memoryStats"`
	Tags          map[string]string `json:"tags"`
	Contexts      []VarzContext         `json:"contexts"`
}

type VarzEmitter struct {
	data map[Identifier]*VarzMetric
	name string
}

type Identifier struct {
	jobName string
	index int
	origin string
}

func NewIdentifier(e *events.Envelope) Identifier{
	jobName := e.GetJob()
	index := e.GetIndex()
	intIndex, err := strconv.Atoi(index)
	if err != nil {
		panic("BOO")
	}

	origin := e.GetOrigin()
	return Identifier{
		jobName: jobName,
		index: intIndex,
		origin: origin,
	}
}

func New(name string) *VarzEmitter {
	return &VarzEmitter{
		data: make(map[Identifier]*VarzMetric),
		name: name,
	}
}

func (e *VarzEmitter) AddMetric(metric *events.Envelope) {
	id := NewIdentifier(metric)
	varzMetric := NewVarzMetric(metric)
	if varzMetric != nil {
		e.data[id] = varzMetric
	}
}

//func (e *VarzEmitter) AlertSlowConsumerError() {
//	tags := make(map[string]interface{})
//
//	ipAddress, err := localip.LocalIP()
//	if err != nil {
//		panic(err)
//	}
//
//	tags["ip"] = ipAddress
//	contextName := "varz-nozzle"
//	name := "slowConsumerAlert"
//	value := uint64(1)
//
//	if _, ok := e.contextMap[contextName]; !ok {
//		e.contextMap[contextName] = contextMetricsMap{Metrics: make(map[string]Metric)}
//	}
//
//	context := e.contextMap[contextName]
//	context.Metrics[name] = Metric{
//		Name:  name,
//		Value: value,
//		Tags:  tags,
//	}
//}

func (e *VarzEmitter) Emit(jobName string, index int) *VarzMessage {
//	e.populateInternalMetrics()

//	contexts := make([]Context, len(e.contextMap))
//	var i = 0
//	for contextName, contextMetricsMap := range e.contextMap {
//		metrics := getMetrics(contextMetricsMap)
//		contexts[i] = Context{Name: contextName, Metrics: metrics}
//		i++
//	}

//	delete(e.contextMap, "varz-nozzle")

//	var memStats runtime.MemStats
//	runtime.ReadMemStats(&memStats)


	return &VarzMessage{
		Contexts:      e.getRelevantContexts(jobName, index),
		Name:          e.name,
		NumCpus:       runtime.NumCPU(),
		NumGoRoutines: runtime.NumGoroutine(),
//		MemoryStats:   mapMemStats(&memStats),
	}
}

func(e *VarzEmitter) getRelevantContexts(job string, index int) []VarzContext {
	contextMap := make(map[string]VarzContext)
	for id, metric := range e.data {
		if id.jobName == job && id.index == index {
			contextName := id.origin
			if _, ok := contextMap[contextName]; !ok {
				contextMap[contextName] = VarzContext{
					Name: contextName,
					Metrics: []VarzMetric{ *metric },
				}
			} else {
				context := contextMap[contextName]
				context.Metrics = append(context.Metrics, *metric)
			}
		}
	}

	contexts := make([]VarzContext, 0)
	for _, context := range contextMap {
		contexts = append(contexts, context)
	}

	return contexts
}

//func (e *VarzEmitter) populateInternalMetrics() {
//	e.ensureVarzNozzleContext()
//	e.ensureSlowConsumerAlertMetric()
//}

//func (e *VarzEmitter) ensureVarzNozzleContext() {
//	_, hasVarzContext := e.contextMap["varz-nozzle"]
//	if !hasVarzContext {
//		e.contextMap["varz-nozzle"] = contextMetricsMap{Metrics: make(map[string]Metric)}
//	}
//}
//
//func (e *VarzEmitter) ensureSlowConsumerAlertMetric() {
//	varzNozzleContext := e.contextMap["varz-nozzle"]
//	_, hasSlowConsumerAlert := varzNozzleContext.Metrics["slowConsumerAlert"]
//	if !hasSlowConsumerAlert {
//		varzNozzleContext.Metrics["slowConsumerAlert"] = defaultSlowConsumerMetric()
//	}
//}

func defaultSlowConsumerMetric() VarzMetric {
	ipAddress, err := localip.LocalIP()
	if err != nil {
		panic(err)
	}

	defaultSlowConsumerMetric := VarzMetric{
		Name:  "slowConsumerAlert",
		Value: 0,
		Tags: map[string]interface{}{
			"ip": ipAddress,
		},
	}
	return defaultSlowConsumerMetric
}
//
//func getMetrics(metricMap contextMetricsMap) []Metric {
//	metrics := make([]Metric, len(metricMap.Metrics))
//	var i = 0
//	for _, metricValue := range metricMap.Metrics {
//		metrics[i] = metricValue
//		i++
//	}
//	return metrics
//}
//
//func mapMemStats(stats *runtime.MemStats) VarzMemoryStats {
//	return VarzMemoryStats{
//		BytesAllocatedHeap:  stats.HeapAlloc,
//		BytesAllocatedStack: stats.StackInuse,
//		BytesAllocated:      stats.Alloc,
//		NumMallocs:          stats.Mallocs,
//		NumFrees:            stats.Frees,
//		LastGCPauseTimeNS:   stats.PauseNs[(stats.NumGC+255)%256],
//	}
//}
