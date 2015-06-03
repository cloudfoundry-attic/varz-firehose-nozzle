package emitter
import (
	"github.com/cloudfoundry/noaa/events"
	"strings"
)

type Emitter struct {
	contextMap map[string]contextMetricsMap
}

type Metric struct {
	Name  string                 `json:"name"`
	Value interface{}            `json:"value"`
	Tags  map[string]interface{} `json:"tags,omitempty"`
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
	Name          string `json:"name"`
	NumCpus       int    `json:"numCPUS"`
	NumGoRoutines int    `json:"numGoRoutines"`
	MemoryStats   VarzMemoryStats   `json:"memoryStats"`
	Tags     map[string]string `json:"tags"`
	Contexts []Context         `json:"contexts"`
}

type contextMetricsMap struct {
	Metrics map[string]Metric
}

func New() *Emitter {
	return &Emitter{
		contextMap: make(map[string]contextMetricsMap),
	}
}

func (e *Emitter) AddMetric(metric *events.Envelope) {
	var name string
	var value interface{}
	tags := make(map[string]interface{})
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

	fields := strings.Split(name, ".")
	var contextName, metricName string
	if len(fields) >= 2 {
		contextName = fields[0]
		metricName = fields[1]
	} else {
		contextName = "default"
		metricName = name
	}

	if _, ok := e.contextMap[contextName]; !ok {
		e.contextMap[contextName] = contextMetricsMap{ Metrics: make(map[string]Metric)}
	}

	context := e.contextMap[contextName]
	context.Metrics[name] = Metric{
		Name: metricName,
		Value: value,
		Tags: tags,
    }

}

func (e *Emitter) Emit() *VarzMessage {
	contexts := make([]Context, len(e.contextMap))
	var i = 0
	for contextName, contextMetricsMap := range(e.contextMap) {
		metrics := getMetrics(contextMetricsMap)
		contexts[i] = Context { Name: contextName, Metrics: metrics}
		i++
	}
	return &VarzMessage{
		Contexts: contexts,
	}
}

func getMetrics(metricMap contextMetricsMap) []Metric {
	metrics := make([]Metric, len(metricMap.Metrics))
	var i = 0
	for _, metricValue := range(metricMap.Metrics) {
		metrics[i] = metricValue
		i++
	}
	return metrics
}

