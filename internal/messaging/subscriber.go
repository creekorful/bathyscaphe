package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// MsgHandler represent an handler for a NATS subscriber
type MsgHandler func(s Subscriber, msg *nats.Msg) error

// Subscriber is something that read msg from an event queue
type Subscriber interface {
	Publisher

	ReadMsg(natsMsg *nats.Msg, msg Msg) error
	QueueSubscribe(subject, queue string, handler MsgHandler) error
	Close()
}

// Subscriber represent a NATS subscriber
type subscriber struct {
	nc *nats.Conn
}

// NewSubscriber create a new subscriber and connect it to given NATS server
func NewSubscriber(address string) (Subscriber, error) {
	nc, err := nats.Connect(address)
	if err != nil {
		return nil, err
	}

	return &subscriber{
		nc: nc,
	}, nil
}

func (s *subscriber) ReadMsg(natsMsg *nats.Msg, msg Msg) error {
	return readJSON(natsMsg, msg)
}

func (s *subscriber) QueueSubscribe(subject, queue string, handler MsgHandler) error {
	// Create the subscriber
	sub, err := s.nc.QueueSubscribeSync(subject, queue)
	if err != nil {
		return err
	}

	for {
		// Read incoming message
		msg, err := sub.NextMsgWithContext(context.Background())
		if err != nil {
			log.Warn().Str("err", err.Error()).Msg("error while reading incoming message, skipping it")
			continue
		}

		// ... And process it
		if err := handler(s, msg); err != nil {
			log.Err(err).Msg("error while processing message")
			continue
		}
	}
}

func (s *subscriber) PublishMsg(msg Msg) error {
	return publishJSON(s.nc, msg.Subject(), msg)
}

func (s *subscriber) Close() {
	s.nc.Close()
}

func readJSON(msg *nats.Msg, body interface{}) error {
	if err := json.Unmarshal(msg.Data, body); err != nil {
		return fmt.Errorf("error while decoding message: %s", err)
	}

	return nil
}
