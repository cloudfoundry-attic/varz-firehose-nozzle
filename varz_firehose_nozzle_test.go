package main_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/apcera/nats"
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/config"
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/emitter"
	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/registrars/collectorregistrar"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/cloudfoundry/yagnats"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var ()

const (
	varzUser = "varzUser"
	varzPass = "varzPass"
	varzPort = 1234
	uaaUser  = "uaaUser"
	uaaPass  = "uaaPass"
	natsPort = 24484
	natsHost = "127.0.0.1"
	natsUser = "nats"
	natsPass = "nats"
	natsType = "MetronAgent"
)

var fakeFirehoseInputChan chan *events.Envelope

var _ = Describe("VarzFirehoseNozzle", func() {
	var (
		nozzleSession *gexec.Session
		natsRunner    *natsrunner.NATSRunner
		fakeUAA       *httptest.Server
		fakeFirehose  *httptest.Server
		configPath    string
	)

	BeforeEach(func() {
		var err error
		fakeFirehoseInputChan = make(chan *events.Envelope)
		natsRunner = natsrunner.NewNATSRunner(natsPort)
		natsRunner.Start()

		fakeUAA = httptest.NewServer(&fakeUAAHandler{})
		fakeFirehose = httptest.NewServer(&fakeFirehoseHandler{})

		configPath = buildConfig(fakeUAA.URL, fakeFirehose.URL)

		nozzleCommand := exec.Command(pathToNozzleExecutable, "-config", configPath)
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
		fakeUAA.Close()
		close(fakeFirehoseInputChan)
		fakeFirehose.Close()
		os.Remove(configPath)
	})

	It("registers itself with collector over NATS", func(done Done) {
		defer close(done)
		message := getNatsMessage(natsRunner.MessageBus)

		Expect(message.Type).To(Equal("MetronAgent"))
		Expect(message.Index).To(Equal(0))
		Expect(message.Host).To(MatchRegexp("[^:]*:[0-9]+"))
		Expect(message.UUID).To(MatchRegexp("[0-9]-[0-9a-f-]{36}"))
		Expect(message.Credentials).To(ConsistOf("varzUser", "varzPass"))
	})

	It("emits messages to the varz-nozzle", func(done Done) {
		defer close(done)
		fakeFirehoseInputChan <- &events.Envelope{
			Origin:    proto.String("origin"),
			Timestamp: proto.Int64(1000000000),
			EventType: events.Envelope_ValueMetric.Enum(),
			ValueMetric: &events.ValueMetric{
				Name:  proto.String("metric.name"),
				Value: proto.Float64(5),
				Unit:  proto.String("gauge"),
			},
			Deployment: proto.String("deployment-name"),
			Job:        proto.String("doppler"),
			Index:      proto.String("0"),
			Ip:         proto.String("127.0.0.1"),
		}

		request, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/varz", varzPort), nil)
		Expect(err).ToNot(HaveOccurred())

		request.SetBasicAuth(varzUser, varzPass)

		resp, err := http.DefaultClient.Do(request)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		jsonBytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		var message emitter.VarzMessage
		err = json.Unmarshal(jsonBytes, &message)
		Expect(err).ToNot(HaveOccurred())
		Expect(message.Contexts).To(HaveLen(1))
		context := message.Contexts[0]
		Expect(context.Name).To(Equal("origin"))
		Expect(context.Metrics).To(HaveLen(1))
		metric := context.Metrics[0]
		Expect(metric.Name).To(Equal("metric.name"))
		Expect(metric.Value).To(BeEquivalentTo(5))
		Expect(metric.Tags).To(HaveKeyWithValue("deployment", "deployment-name"))
		Expect(metric.Tags).To(HaveKeyWithValue("job", "doppler"))
		Expect(metric.Tags).To(HaveKeyWithValue("index", "0"))
		Expect(metric.Tags).To(HaveKeyWithValue("ip", "127.0.0.1"))

	})
})

func buildConfig(uaaURL string, firehoseURL string) string {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "varz_nozzle_config")
	if err != nil {
		panic(err)
	}

	config := config.VarzConfig{
		UAAURL:               uaaURL,
		TrafficControllerURL: strings.Replace(firehoseURL, "http", "ws", 1),
		UAAUser:              uaaUser,
		UAAPass:              uaaPass,
		Config: cfcomponent.Config{
			CollectorRegistrarIntervalMilliseconds: 100,
			VarzPort:  varzPort,
			VarzUser:  varzUser,
			VarzPass:  varzPass,
			NatsHosts: []string{natsHost},
			NatsPort:  natsPort,
			NatsUser:  natsUser,
			NatsPass:  natsPass,
		},
		NatsType: natsType,
	}

	jsonBytes, err := json.Marshal(&config)
	_, err = tmpFile.Write(jsonBytes)

	if err != nil {
		panic(err)
	}

	tmpFile.Close()
	return tmpFile.Name()
}

type natsMessage struct {
	Type        string   `json:"type"`
	Index       int      `json:"index"`
	Host        string   `json:"host"`
	UUID        string   `json:"uuid"`
	Credentials []string `json:"credentials"`
}

func getNatsMessage(natsClient yagnats.NATSConn) natsMessage {
	messageChan := make(chan []byte)
	natsClient.Subscribe(collectorregistrar.AnnounceComponentMessageSubject, func(msg *nats.Msg) {
		messageChan <- msg.Data
	})

	var message natsMessage
	messageBytes := <-messageChan
	err := json.Unmarshal(messageBytes, &message)
	if err != nil {
		panic(err)
	}

	return message
}

type fakeUAAHandler struct{}

func (f *fakeUAAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Write([]byte(`
		{
			"token_type": "bearer",
			"access_token": "good-token"
		}
	`))
}

type fakeFirehoseHandler struct{}

func (f *fakeFirehoseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("Authorization")

	if authorization != "bearer good-token" {
		log.Printf("Bad token passed to firehose: %s", authorization)
		w.WriteHeader(http.StatusUnauthorized)
		r.Body.Close()
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	ws, _ := upgrader.Upgrade(w, r, nil)

	defer ws.Close()
	defer ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Time{})

	for envelope := range fakeFirehoseInputChan {
		buffer, err := proto.Marshal(envelope)
		Expect(err).NotTo(HaveOccurred())
		err = ws.WriteMessage(websocket.BinaryMessage, buffer)
		Expect(err).NotTo(HaveOccurred())
	}
}
