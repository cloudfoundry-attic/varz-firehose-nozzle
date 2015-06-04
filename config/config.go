package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
)

type VarzConfig struct {
	cfcomponent.Config
	Index                  uint
	UAAURL                 string
	UAAUser                string
	UAAPass                string
	NatsType               string
	InsecureSSLSkipVerify  bool
	TrafficControllerURL   string
	FirehoseSubscriptionID string
}

func ParseConfig(configPath string) (*VarzConfig, error) {
	configBytes, err := ioutil.ReadFile(configPath)
	var config VarzConfig
	if err != nil {
		return nil, fmt.Errorf("Can not read config file [%s]: %s", configPath, err)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, fmt.Errorf("Can not parse config file %s: %s", configPath, err)
	}
	return &config, err
}
