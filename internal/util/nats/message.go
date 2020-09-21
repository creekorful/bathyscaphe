package nats

type Msg interface {
	Subject() string
}
