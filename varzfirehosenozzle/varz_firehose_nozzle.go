package varzfirehosenozzle

import (
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gorilla/websocket"
)

type FirehoseConsumer interface {
	Firehose(string, string, chan<- *events.Envelope, chan<- error)
	Close() error
}

type Emitter interface {
	AddMetric(env *events.Envelope)
	AlertSlowConsumerError()
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
			n.handleMessage(envelope)
		case err, ok := <-errs:
			if !ok {
				n.consumer.Close()
				return
			}
			n.handleError(err)
		}
	}
}

func (n *VarzFirehoseNozzle) handleMessage(message *events.Envelope) {
	n.varzEmitter.AddMetric(message)
	if n.isSlowConsumerAlert(message) && message.CounterEvent.GetDelta() != 0 {
		n.logger.Warn("We've intercepted an upstream message which indicates that the nozzle or the TrafficController is not keeping up. Please try scaling up the nozzle.")
		n.varzEmitter.AlertSlowConsumerError()
	}
}

func (n *VarzFirehoseNozzle) isSlowConsumerAlert(message *events.Envelope) bool {
	return message.GetEventType() == events.Envelope_CounterEvent && message.CounterEvent.GetName() == "TruncatingBuffer.DroppedMessages"
}

func (n *VarzFirehoseNozzle) handleError(err error) {
	n.logger.Errorf("Error while reading from the firehose: %s", err.Error())
	if isCloseError(err) {
		n.logger.Warn("Disconnected because nozzle couldn't keep up. Please try scaling up the nozzle.")
		n.varzEmitter.AlertSlowConsumerError()
	}
}

func isCloseError(err error) bool {
	switch closeErr := err.(type) {
	case *websocket.CloseError:
		return closeErr.Code == websocket.ClosePolicyViolation
	}
	return false
}
