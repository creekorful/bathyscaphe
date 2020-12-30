package event

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
)

// Publisher is something that push an event
type Publisher interface {
	PublishEvent(event Event) error
	PublishJSON(exchange string, msg RawMessage) error
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

func (p *publisher) PublishEvent(event Event) error {
	evtBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error while encoding event: %s", err)
	}

	return p.PublishJSON(event.Exchange(), RawMessage{Body: evtBytes})
}

func (p *publisher) PublishJSON(exchange string, msg RawMessage) error {
	return p.channel.Publish(exchange, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         msg.Body,
		DeliveryMode: amqp.Persistent,
		Headers:      msg.Headers,
	})
}

func (p *publisher) Close() error {
	return p.channel.Close()
}
