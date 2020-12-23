package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
	"io"
)

// Handler represent an event handler
type Handler func(Subscriber, io.Reader) error

// Subscriber is something that read msg from an event queue
type Subscriber interface {
	Publisher

	Read(body io.Reader, event Event) error
	SubscribeAsync(exchange, queue string, handler Handler) error
}

// Subscriber represent a subscriber
type subscriber struct {
	channel *amqp.Channel
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
		channel: c,
	}, nil
}

func (s *subscriber) PublishEvent(event Event) error {
	evtBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error while encoding event: %s", err)
	}

	return s.PublishJSON(event.Exchange(), evtBytes)
}

func (p *subscriber) PublishJSON(exchange string, event []byte) error {
	return p.channel.Publish(exchange, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         event,
		DeliveryMode: amqp.Persistent,
	})
}

func (s *subscriber) Close() error {
	return s.channel.Close()
}

func (s *subscriber) Read(body io.Reader, event Event) error {
	return readJSON(body, event)
}

func (s *subscriber) SubscribeAsync(exchange, queue string, handler Handler) error {
	// First of all declare the exchange
	if err := s.channel.ExchangeDeclare(exchange, amqp.ExchangeFanout, true, false, false, false, nil); err != nil {
		return err
	}

	// Then declare the queue
	q, err := s.channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	// Bind the queue to the exchange
	if err := s.channel.QueueBind(q.Name, "", exchange, false, nil); err != nil {
		return err
	}

	// Start consuming asynchronously
	deliveries, err := s.channel.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for delivery := range deliveries {
			if err := handler(s, bytes.NewReader(delivery.Body)); err != nil {
				log.Err(err).Msg("error while processing event")
			}

			// Ack no matter what happen since we doesn't care about failing event (yet?)
			if err := delivery.Ack(false); err != nil {
				log.Err(err).Msg("error while acknowledging event")
			}
		}
	}()

	return nil
}

func readJSON(body io.Reader, event interface{}) error {
	if err := json.NewDecoder(body).Decode(event); err != nil {
		return fmt.Errorf("error while decoding event: %s", err)
	}

	return nil
}
