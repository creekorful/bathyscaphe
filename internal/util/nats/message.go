package nats

// Msg represent a message send-able trough NATS
type Msg interface {
	// Subject returns the subject where message should be push
	Subject() string
}
