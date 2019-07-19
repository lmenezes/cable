package cable

// bufferSize is the number of messages that can be enqueued in the inbox and
// outbox channels
const DefaultBufferSize = 100

// Pump is a struct that describes am entity with an inbox and
// and outbox channel of Messages
type Pump struct {
	Inbox  chan Message
	Outbox chan Message
}

func newPump() *Pump {
	return &Pump{
		Inbox:  make(chan Message, DefaultBufferSize),
		Outbox: make(chan Message, DefaultBufferSize),
	}
}
