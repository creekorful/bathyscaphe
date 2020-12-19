package messaging

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
)

// Publisher is something that push msg to an event queue
type Publisher interface {
	PublishMsg(msg Msg) error
	Close()
}

type publisher struct {
	rc *amqp.Channel
}

// NewPublisher create a new Publisher instance
func NewPublisher(amqpURI string) (Publisher, error) {
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		return nil, err
	}

	c, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &publisher{
		rc: c,
	}, nil
}

func (p *publisher) PublishMsg(msg Msg) error {
	return publishJSON(p.rc, msg.Subject(), msg)
}

func (p *publisher) Close() {
	_ = p.rc.Close()
}

func publishJSON(rc *amqp.Channel, subject string, msg interface{}) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error while encoding message: %s", err)
	}

	return rc.Publish("", subject, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         msgBytes,
		DeliveryMode: amqp.Persistent,
	})
}
