package event

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
)

// Publisher is something that push an event
type Publisher interface {
	Publish(event Event) error
	Close() error
}

type publisher struct {
	channel *amqp.Channel
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
		channel: c,
	}, nil
}

func (p *publisher) Publish(event Event) error {
	return publishJSON(p.channel, event.Exchange(), event)
}

func (p *publisher) Close() error {
	return p.channel.Close()
}

func publishJSON(rc *amqp.Channel, exchange string, event interface{}) error {
	evtBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error while encoding event: %s", err)
	}

	return rc.Publish(exchange, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         evtBytes,
		DeliveryMode: amqp.Persistent,
	})
}
