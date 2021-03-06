package emitter

import (
	"runtime"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/pivotal-golang/localip"
	"strconv"
)

type VarzEmitter struct {
	contextMap map[string]contextMetricsMap
	name       string
	job        string
	index      uint
	deployment string
}

type Metric struct {
	Name  string            `json:"name"`
	Value interface{}       `json:"value"`
	Tags  map[string]string `json:"tags,omitempty"`
}

type metricKey struct {
	name       string
	deployment string
	job        string
	index      string
	ip         string
}

type Context struct {
	Name    string   `json:"name"`
	Metrics []Metric `json:"metrics"`
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
	Contexts      []Context         `json:"contexts"`
}

type contextMetricsMap struct {
	Metrics map[metricKey]Metric
}

func New(name string, job string, index uint, deployment string) *VarzEmitter {
	return &VarzEmitter{
		contextMap: make(map[string]contextMetricsMap),
		name:       name,
		job:        job,
		index:      index,
		deployment: deployment,
	}
}

func (e *VarzEmitter) AddMetric(metric *events.Envelope) {
	var name string
	var value interface{}
	tags := make(map[string]string)
	switch metric.GetEventType() {
	case events.Envelope_ValueMetric:
		name = metric.GetValueMetric().GetName()
		value = metric.GetValueMetric().GetValue()
	case events.Envelope_CounterEvent:
		name = metric.GetCounterEvent().GetName()
		value = metric.GetCounterEvent().GetTotal()
	default:
		return
	}

	tags["deployment"] = metric.GetDeployment()
	tags["ip"] = metric.GetIp()
	tags["job"] = metric.GetJob()
	tags["index"] = metric.GetIndex()

	contextName := metric.GetOrigin()

	if _, ok := e.contextMap[contextName]; !ok {
		e.contextMap[contextName] = contextMetricsMap{Metrics: make(map[metricKey]Metric)}
	}

	key := metricKey{
		name:       name,
		deployment: metric.GetDeployment(),
		job:        metric.GetJob(),
		index:      metric.GetIndex(),
		ip:         metric.GetIp(),
	}

	context := e.contextMap[contextName]
	context.Metrics[key] = Metric{
		Name:  name,
		Value: value,
		Tags:  tags,
	}

}

func (e *VarzEmitter) AlertSlowConsumerError() {
	tags := make(map[string]string)

	ipAddress, err := localip.LocalIP()
	if err != nil {
		panic(err)
	}

	tags["ip"] = ipAddress
	tags["deployment"] = e.deployment
	tags["job"] = e.job
	tags["index"] = strconv.Itoa(int(e.index))

	contextName := "varz-nozzle"
	name := "slowConsumerAlert"
	value := uint64(1)

	if _, ok := e.contextMap[contextName]; !ok {
		e.contextMap[contextName] = contextMetricsMap{Metrics: make(map[metricKey]Metric)}
	}

	context := e.contextMap[contextName]

	key := metricKey{
		name: name,
		ip:   ipAddress,
	}

	context.Metrics[key] = Metric{
		Name:  name,
		Value: value,
		Tags:  tags,
	}
}

func (e *VarzEmitter) Emit() *VarzMessage {
	e.populateInternalMetrics()

	contexts := make([]Context, len(e.contextMap))
	var i = 0
	for contextName, contextMetricsMap := range e.contextMap {
		metrics := getMetrics(contextMetricsMap)
		contexts[i] = Context{Name: contextName, Metrics: metrics}
		i++
	}

	delete(e.contextMap, "varz-nozzle")

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &VarzMessage{
		Contexts:      contexts,
		Name:          e.name,
		NumCpus:       runtime.NumCPU(),
		NumGoRoutines: runtime.NumGoroutine(),
		MemoryStats:   mapMemStats(&memStats),
		Tags: map[string]string{
			"deployment": e.deployment,
			"job":        e.job,
			"index":      strconv.Itoa(int(e.index)),
		},
	}
}

func (e *VarzEmitter) populateInternalMetrics() {
	e.ensureVarzNozzleContext()
	e.ensureSlowConsumerAlertMetric()
}

func (e *VarzEmitter) ensureVarzNozzleContext() {
	_, hasVarzContext := e.contextMap["varz-nozzle"]
	if !hasVarzContext {
		e.contextMap["varz-nozzle"] = contextMetricsMap{Metrics: make(map[metricKey]Metric)}
	}
}

func (e *VarzEmitter) ensureSlowConsumerAlertMetric() {
	varzNozzleContext := e.contextMap["varz-nozzle"]

	ipAddress, err := localip.LocalIP()
	if err != nil {
		panic(err)
	}

	key := metricKey{
		name: "slowConsumerAlert",
		ip:   ipAddress,
	}

	defaultSlowConsumerMetric := Metric{
		Name:  "slowConsumerAlert",
		Value: 0,
		Tags: map[string]string{
			"ip":         ipAddress,
			"deployment": e.deployment,
			"job":        e.job,
			"index":      strconv.Itoa(int(e.index)),
		},
	}

	_, hasSlowConsumerAlert := varzNozzleContext.Metrics[key]
	if !hasSlowConsumerAlert {
		varzNozzleContext.Metrics[key] = defaultSlowConsumerMetric
	}
}

func getMetrics(metricMap contextMetricsMap) []Metric {
	metrics := make([]Metric, len(metricMap.Metrics))
	var i = 0
	for _, metricValue := range metricMap.Metrics {
		metrics[i] = metricValue
		i++
	}
	return metrics
}

func mapMemStats(stats *runtime.MemStats) VarzMemoryStats {
	return VarzMemoryStats{
		BytesAllocatedHeap:  stats.HeapAlloc,
		BytesAllocatedStack: stats.StackInuse,
		BytesAllocated:      stats.Alloc,
		NumMallocs:          stats.Mallocs,
		NumFrees:            stats.Frees,
		LastGCPauseTimeNS:   stats.PauseNs[(stats.NumGC+255)%256],
	}
}
