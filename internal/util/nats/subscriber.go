package nats

import (
	"context"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// MsgHandler represent an handler for a NATS subscriber
type MsgHandler func(nc *nats.Conn, msg *nats.Msg) error

// Subscriber represent a NATS subscriber
type Subscriber struct {
	nc *nats.Conn
}

// NewSubscriber create a new subscriber and connect it to given NATS server
func NewSubscriber(address string) (*Subscriber, error) {
	nc, err := nats.Connect(address)
	if err != nil {
		log.Err(err).Str("server-uri", address).Msg("Error while connecting to NATS server")
		return nil, err
	}

	return &Subscriber{
		nc: nc,
	}, nil
}

// QueueSubscribe subscribe to given subject, with given queue
func (qs *Subscriber) QueueSubscribe(subject, queue string, handler MsgHandler) error {
	// Create the subscriber
	sub, err := qs.nc.QueueSubscribeSync(subject, queue)
	if err != nil {
		log.Err(err).Msg("Error while reading message from NATS server")
		return err
	}

	for {
		// Read incoming message
		msg, err := sub.NextMsgWithContext(context.Background())
		if err != nil {
			log.Warn().Str("err", err.Error()).Msg("Skipping current message because of error")
			continue
		}

		// ... And process it
		if err := handler(qs.nc, msg); err != nil {
			log.Warn().Str("error", err.Error()).Msg("Skipping current message because of error")
			continue
		}
	}
}

// Close terminate the connection to the NATS server
func (qs *Subscriber) Close() {
	qs.nc.Close()
}
