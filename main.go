package main

import (
	"crypto/tls"
	"flag"
	"github.com/cloudfoundry-incubator/uaago"
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/config"
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/emitter"
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/server"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/registrars/collectorregistrar"
	"github.com/cloudfoundry/noaa"
	"github.com/cloudfoundry/noaa/events"
	"os"
	"time"
)

type varzHealthMonitor struct{}

func (*varzHealthMonitor) Ok() bool {
	return true
}

func main() {
	var (
		configFilePath = flag.String("config", "config/varz_firehose_nozzle.json", "Location of the nozzle config json file")
	)
	flag.Parse()

	logger := initLogger()

	config, err := config.ParseConfig(*configFilePath)
	if err != nil {
		logger.Fatal(err.Error())
		return
	}

	uaaClient, err := uaago.NewClient(config.UAAURL)
	if err != nil {
		logger.Fatalf("Error creating uaa client: %s", err.Error())
		return
	}

	authToken, err := uaaClient.GetAuthToken(config.UAAUser, config.UAAPass, config.InsecureSSLSkipVerify)
	if err != nil {
		logger.Fatalf("Error getting oauth token: %s. Please check your username and password.", err.Error())
		return
	}

	consumer := noaa.NewConsumer(
		config.TrafficControllerURL,
		&tls.Config{InsecureSkipVerify: config.InsecureSSLSkipVerify},
		nil)
	messages := make(chan *events.Envelope)
	errs := make(chan error)
	done := make(chan struct{})
	go consumer.Firehose(config.FireshoseSubscriptionID, authToken, messages, errs, done)

	go func() {
		err := <-errs
		logger.Errorf("Error while reading from the firehose: %s", err.Error())
		close(done)
	}()

	registrar, err := initRegistrar(config, logger)
	if err != nil {
		logger.Fatal(err.Error())
		return
	}

	varzEmitter := emitter.New("varz-nozzle")
	varzServer := server.New(varzEmitter, int(config.VarzPort), config.VarzUser, config.VarzPass)

	go varzServer.Start()
	go registrar.Run()

	for {
		select {
		case envelope := <-messages:
			varzEmitter.AddMetric(envelope)
		case <-done:
			consumer.Close()
		}
	}
}

func initLogger() *gosteno.Logger {
	c := &gosteno.Config{
		Sinks: []gosteno.Sink{
			gosteno.NewIOSink(os.Stdout),
		},
		Level:     gosteno.LOG_INFO,
		Codec:     gosteno.NewJsonCodec(),
		EnableLOC: true,
	}
	gosteno.Init(c)

	return gosteno.NewLogger("Varz Firehose Nozzle")
}

func initRegistrar(config *config.VarzConfig, logger *gosteno.Logger) (*collectorregistrar.CollectorRegistrar, error) {
	interval := time.Duration(config.CollectorRegistrarIntervalMilliseconds) * time.Millisecond
	instrumentables := []instrumentation.Instrumentable{}
	component, err := cfcomponent.NewComponent(logger, config.NatsType, config.Index, &varzHealthMonitor{}, config.VarzPort, []string{config.VarzUser, config.VarzPass}, instrumentables)

	if err != nil {
		return nil, err
	}

	registrar := collectorregistrar.NewCollectorRegistrar(cfcomponent.DefaultYagnatsClientProvider, component, interval, &config.Config)

	return registrar, nil
}
