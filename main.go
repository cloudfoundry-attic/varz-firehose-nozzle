package main

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/registrars/collectorregistrar"
	"io/ioutil"
	"time"
	"flag"
	"os"
)

type varzConfig struct {
	cfcomponent.Config
	Index uint
}

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

	config, err := parseConfig(*configFilePath)
	if err != nil {
		logger.Fatal(err.Error())
		return
	}

	registrar, err := initRegistrar(config, logger)
	if err != nil {
		logger.Fatal(err.Error())
		return
	}

	registrar.Run()
}

func parseConfig(configPath string) (*varzConfig, error) {
	configBytes, err := ioutil.ReadFile(configPath)
	var config varzConfig
	if err != nil {
		return nil, fmt.Errorf("Can not read config file [%s]: %s", configPath, err)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, fmt.Errorf("Can not parse config file %s: %s", configPath, err)
	}
	return &config, err
}

func initLogger() *gosteno.Logger{
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

func initRegistrar(config *varzConfig, logger *gosteno.Logger) (*collectorregistrar.CollectorRegistrar, error) {
	interval := time.Duration(config.CollectorRegistrarIntervalMilliseconds) * time.Millisecond
	instrumentables := []instrumentation.Instrumentable{}
	component, err := cfcomponent.NewComponent(logger, "MetronAgent", config.Index, &varzHealthMonitor{}, config.VarzPort, []string{config.VarzUser, config.VarzPass}, instrumentables)

	if err != nil {
		return nil, err
	}

	registrar := collectorregistrar.NewCollectorRegistrar(cfcomponent.DefaultYagnatsClientProvider, component, interval, &config.Config)

	return registrar, nil
}
