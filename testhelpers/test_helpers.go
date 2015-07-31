package testhelpers

import "github.com/cloudfoundry-incubator/varz-firehose-nozzle/emitter"

const VarzSlowConsumerContext = 1

func FindContext(contextName string, contexts []emitter.Context) *emitter.Context {
	for _, c := range contexts {
		if c.Name == contextName {
			return &c
		}
	}
	return nil
}
