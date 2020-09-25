package messaging

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
)

// Publisher is something that push msg to an event queue
type Publisher interface {
	PublishMsg(msg Msg) error
	Close()
}

type publisher struct {
	nc *nats.Conn
}

// NewPublisher create a new Publisher instance
func NewPublisher(natsURI string) (Publisher, error) {
	nc, err := nats.Connect(natsURI)
	if err != nil {
		return nil, err
	}

	return &publisher{
		nc: nc,
	}, nil
}

func (p *publisher) PublishMsg(msg Msg) error {
	return publishJSON(p.nc, msg.Subject(), msg)
}

func (p *publisher) Close() {
	p.nc.Close()
}

func publishJSON(nc *nats.Conn, subject string, msg interface{}) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error while encoding message: %s", err)
	}

	return nc.Publish(subject, msgBytes)
}
