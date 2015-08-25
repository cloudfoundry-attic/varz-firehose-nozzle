package server_test

import (
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/server"

	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/emitter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("VarzServer", func() {
	var varzServer *server.VarzServer
	const varzPort = 8888
	const varzUser = "good-user"
	const varzPass = "good-pass"

	It("emits expected JSON on varz endpoint", func() {
		emitter := &fakeEmitter{
			message: buildVarzMessage(),
		}
		varzServer = server.New(emitter, varzPort, varzUser, varzPass)

		response := httptest.NewRecorder()
		url := fmt.Sprintf("http://localhost:%d", varzPort)
		request, _ := http.NewRequest("GET", url, nil)
		request.SetBasicAuth(varzUser, varzPass)

		varzServer.ServeHTTP(response, request)
		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(response.Body.String()).To(MatchJSON(`{"name":"fake-message","numCPUS":2,"numGoRoutines":2,"memoryStats":{"numBytesAllocatedHeap":0,"numBytesAllocatedStack":0,"numBytesAllocated":0,"numMallocs":0,"numFrees":0,"lastGCPauseTimeNS":0},"tags":null,"contexts":[{"name":"contextName1","metrics":[{"name":"metricName","value":10,"tags":{"deployment":"our-deployment","index":"0","ip":"192.168.0.1","job":"doppler"}}]}]}`))
	})

	It("return error with incorrect user or password", func() {
		emitter := &fakeEmitter{}
		varzServer = server.New(emitter, varzPort, varzUser, varzPass)

		response := httptest.NewRecorder()
		url := fmt.Sprintf("http://localhost:%d", varzPort)
		request, _ := http.NewRequest("GET", url, nil)
		request.SetBasicAuth("bad-user", "bad-password")

		varzServer.ServeHTTP(response, request)
		Expect(response.Code).To(Equal(http.StatusUnauthorized))
	})
})

type fakeEmitter struct {
	message *emitter.VarzMessage
}

func (f *fakeEmitter) Emit() *emitter.VarzMessage {
	return f.message
}

func buildVarzMessage() *emitter.VarzMessage {
	memoryStats := emitter.VarzMemoryStats{}
	contexts := []emitter.Context{
		{
			Name: "contextName1",
			Metrics: []emitter.Metric{
				{
					Name:  "metricName",
					Value: uint64(10),
					Tags: map[string]string{
						"ip":         "192.168.0.1",
						"job":        "doppler",
						"index":      "0",
						"deployment": "our-deployment",
					},
				},
			},
		},
	}
	return &emitter.VarzMessage{
		Name:          "fake-message",
		NumCpus:       2,
		NumGoRoutines: 2,
		MemoryStats:   memoryStats,
		Contexts:      contexts,
	}
}
