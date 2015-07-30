package varzfirehosenozzle

import (
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/sonde-go/events"
)

type FirehoseConsumer interface {
	Firehose(string, string, chan<- *events.Envelope, chan<- error)
	Close() error
}

type Emitter interface {
	AddMetric(env *events.Envelope)
}

type VarzFirehoseNozzle struct {
	subscriptionID string
	authToken      string
	consumer       FirehoseConsumer
	varzEmitter    Emitter
	logger         *gosteno.Logger
}

func NewVarzFirehoseNozzle(subscriptionID string, authToken string, firehose FirehoseConsumer, varzEmitter Emitter, logger *gosteno.Logger) *VarzFirehoseNozzle {
	return &VarzFirehoseNozzle{
		subscriptionID: subscriptionID,
		authToken:      authToken,
		consumer:       firehose,
		varzEmitter:    varzEmitter,
		logger:         logger,
	}
}

func (n *VarzFirehoseNozzle) Run() {
	messages := make(chan *events.Envelope)
	errs := make(chan error)
	go n.consumer.Firehose(n.subscriptionID, n.authToken, messages, errs)

	for {
		select {
		case envelope, ok := <-messages:
			if !ok {
				n.consumer.Close()
				return
			}
			n.varzEmitter.AddMetric(envelope)
		case err, ok := <-errs:
			if !ok {
				n.consumer.Close()
				return
			}
			n.logger.Errorf("Error while reading from the firehose: %s", err.Error())
		}
	}
}
