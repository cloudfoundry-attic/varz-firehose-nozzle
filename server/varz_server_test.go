package server_test

import (
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry/noaa/events"
	"github.com/gogo/protobuf/proto"
	"net/http/httptest"
	"net/http"
	"fmt"
)

var _ = Describe("VarzServer", func() {
	var varzServer *server.VarzServer
	const varzPort = 8888
	const varzUser = "good-user"
	const varzPass = "good-pass"

	It("emits expected JSON on varz endpoint", func() {
		varzServer = server.New(varzPort, varzUser, varzPass)

		envelope := &events.Envelope{
			Origin: proto.String("fake-origin"),
			EventType: events.Envelope_ValueMetric.Enum(),
		}

		varzServer.Emit(envelope)

		response := httptest.NewRecorder()
		url := fmt.Sprintf("http://localhost:%d", varzPort)
		request, _ := http.NewRequest("GET", url, nil)

		varzServer.ServeHTTP(response, request)
		Expect(response.Body.String()).To(MatchJSON("{}"))

	})
})
