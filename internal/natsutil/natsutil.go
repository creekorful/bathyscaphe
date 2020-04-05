package natsutil

import (
	"encoding/json"
	"github.com/nats-io/nats.go"
)

// PublishJson publish given message serialized in json with given subject
func PublishJson(nc *nats.Conn, subject string, msg interface{}) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return nc.Publish(subject, msgBytes)
}
