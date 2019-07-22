package cable

import log "github.com/sirupsen/logrus"

// DefaultBufferSize is the number of messages that can be enqueued in the inbox and
// outbox channels
const DefaultBufferSize = 100

// Pump is a struct that describes am entity with an inbox and
// and outbox channel of Messages
type Pump struct {
	Inbox  chan Message
	Outbox chan Message
}

// NewPump returns the address of a new value of the Pump struct with
// Inbox and Outbox as buffered channels of size DefaultBufferSize
func NewPump() *Pump {
	return &Pump{
		Inbox:  make(chan Message, DefaultBufferSize),
		Outbox: make(chan Message, DefaultBufferSize),
	}
}

// BidirectionalPumpConnection defines a connection between two
// pipes such as the messages arriving at the inbox of one of them
// are relayed to the outbox of the other and vice-versa
type BidirectionalPumpConnection struct {
	Left  *Pump
	Right *Pump
	stop  chan interface{}
}

// NewBidirectionalPumpConnection returns the address of a new
// BidirectionalPumpConnection
func NewBidirectionalPumpConnection(left *Pump, right *Pump) *BidirectionalPumpConnection {
	return &BidirectionalPumpConnection{
		Left:  left,
		Right: right,
		stop:  make(chan interface{}),
	}
}

// Go spawns a goroutine processing the bidirectional connection
func (c BidirectionalPumpConnection) Go() {
	go func() {
		for {
			select {
			case m := <-c.Left.Inbox:
				log.Debugf("[%+v]: %s", c.Left, m)
				c.Right.Outbox <- m
			case m := <-c.Right.Inbox:
				log.Debugf("[%+v]: %s", c.Right, m)
				c.Left.Outbox <- m
			case <-c.stop:
				// TODO: stop left and right pumps
				return
			}
		}
	}()
}

// Stop stops the processing of the bidirectional connection
func (c BidirectionalPumpConnection) Stop() {
	c.stop <- true
}
