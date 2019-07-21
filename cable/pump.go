package cable

// DefaultBufferSize is the number of messages that can be enqueued in the inbox and
// outbox channels
const DefaultBufferSize = 100

// Pump is a struct that describes am entity with an inbox and
// and outbox channel of Messages
//
// TODO: Implement a stop mechanism for pumps, and avoid races.
//		Currently, pumps in both client and telegram are infinite loops.
//		They should have a way to stop manually.
//
type Pump struct {
	Inbox  chan Message
	Outbox chan Message
}

// NewPump creates a new value of the Pump struct with Inbox and Outbox as
// buffered channels of size DefaultBufferSize
func NewPump() *Pump {
	return &Pump{
		Inbox:  make(chan Message, DefaultBufferSize),
		Outbox: make(chan Message, DefaultBufferSize),
	}
}
