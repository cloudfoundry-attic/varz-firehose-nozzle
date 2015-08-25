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
	Job                    string
	Deployment             string
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
		return nil, fmt.Errorf("cannot read config file [%s]: %s", configPath, err)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, fmt.Errorf("cannot parse config file %s: %s", configPath, err)
	}

	if config.Deployment == "" {
		return nil, fmt.Errorf("missing field: deployment")
	}

	if config.Job == "" {
		return nil, fmt.Errorf("missing field: job")
	}

	return &config, err
}
