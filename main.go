package main

import (
	"crypto/tls"
	"flag"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/uaago"
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/config"
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/emitter"
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/server"
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/varzfirehosenozzle"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/registrars/collectorregistrar"
	"github.com/cloudfoundry/noaa"
	"os/signal"
	"runtime/pprof"
	"syscall"
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

	component, registrar, err := initRegistrar(config, logger)
	if err != nil {
		logger.Fatal(err.Error())
		return
	}

	threadDumpChan := registerGoRoutineDumpSignalChannel()
	defer close(threadDumpChan)
	go dumpGoRoutine(threadDumpChan)

	varzEmitter := emitter.New("varz-nozzle", config.Job, config.Index, config.Deployment)

	varzServer := server.New(varzEmitter, int(component.StatusPort), component.StatusCredentials[0], component.StatusCredentials[1])
	go varzServer.Start()
	go registrar.Run()

	logger.Info("Started listening for messages")
	defer logger.Info("Exiting")

	firehoseNozzle := varzfirehosenozzle.NewVarzFirehoseNozzle(config.FirehoseSubscriptionID, authToken, consumer, varzEmitter, logger)
	firehoseNozzle.Run()
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

func initRegistrar(config *config.VarzConfig, logger *gosteno.Logger) (*cfcomponent.Component, *collectorregistrar.CollectorRegistrar, error) {
	interval := time.Duration(config.CollectorRegistrarIntervalMilliseconds) * time.Millisecond
	instrumentables := []instrumentation.Instrumentable{}
	component, err := cfcomponent.NewComponent(logger, config.NatsType, config.Index, &varzHealthMonitor{}, config.VarzPort, []string{config.VarzUser, config.VarzPass}, instrumentables)

	if err != nil {
		return nil, nil, err
	}

	registrar := collectorregistrar.NewCollectorRegistrar(cfcomponent.DefaultYagnatsClientProvider, component, interval, &config.Config)

	return &component, registrar, nil
}

func registerGoRoutineDumpSignalChannel() chan os.Signal {
	threadDumpChan := make(chan os.Signal, 1)
	signal.Notify(threadDumpChan, syscall.SIGUSR1)

	return threadDumpChan
}

func dumpGoRoutine(dumpChan chan os.Signal) {
	for range dumpChan {
		goRoutineProfiles := pprof.Lookup("goroutine")
		if goRoutineProfiles != nil {
			goRoutineProfiles.WriteTo(os.Stdout, 2)
		}
	}
}
