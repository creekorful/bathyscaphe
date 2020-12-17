package messaging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
	"io"
)

// MsgHandler represent an handler for a subscriber
type MsgHandler func(s Subscriber, body io.Reader) error

// Subscriber is something that read msg from an event queue
type Subscriber interface {
	Publisher

	ReadMsg(body io.Reader, msg Msg) error
	QueueSubscribe(subject, queue string, handler MsgHandler) error
	Close()
}

// Subscriber represent a subscriber
type subscriber struct {
	rc *amqp.Channel
}

// NewSubscriber create a new subscriber and connect it to given server
func NewSubscriber(amqpURI string) (Subscriber, error) {
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		return nil, err
	}

	c, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := c.Qos(1, 0, false); err != nil {
		return nil, err
	}

	return &subscriber{
		rc: c,
	}, nil
}

func (s *subscriber) ReadMsg(body io.Reader, msg Msg) error {
	return readJSON(body, msg)
}

func (s *subscriber) QueueSubscribe(subject, queue string, handler MsgHandler) error {
	q, err := s.rc.QueueDeclare(subject, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error while declaring queue: %s", err)
	}

	deliveries, err := s.rc.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error while consuming queue: %s", err)
	}

	for d := range deliveries {
		if err := handler(s, bytes.NewReader(d.Body)); err != nil {
			log.Err(err).Msg("error while processing message")
		}

		// Ack no matter what since we doesn't care about failing messages
		if err := d.Ack(false); err != nil {
			log.Err(err).Msg("error while ack`ing message")
		}
	}

	return nil
}

func (s *subscriber) PublishMsg(msg Msg) error {
	return publishJSON(s.rc, msg.Subject(), msg)
}

func (s *subscriber) Close() {
	_ = s.rc.Close()
}

func readJSON(body io.Reader, where interface{}) error {
	if err := json.NewDecoder(body).Decode(where); err != nil {
		return fmt.Errorf("error while decoding message: %s", err)
	}

	return nil
}
