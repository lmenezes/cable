package cable

import log "github.com/sirupsen/logrus"

// DefaultBufferSize is the number of messages that can be enqueued in the inbox and
// outbox channels
const DefaultBufferSize = 100

// Pumper is a composed interface implemented by Read and Write pumpers
type Pumper interface {
	ReadPumper
	WritePumper
}

// ReadPumper is the interface implemented by Read pumpers.
// Read pumpers process events and feed them into the Inbox
type ReadPumper interface {
	// GoRead spawns a new goroutine to read messages and feed them into
	// inbox
	GoRead()
	// StopRead stops the read goroutine
	StopRead()
	// Inbox returns a channel containing the messages read by the ReadPumper
	Inbox() chan Message
}

// WritePumper is the interface implemented by Write pumpers.
// Write pumpers read events fed into the outbox and write them somewhere else
type WritePumper interface {
	// GoRead spawns a new goroutine to write messages arriving at the outbox
	GoWrite()
	// StopWrite stops the write goroutine
	StopWrite()
	// Outbox returns a channel of messages, which will be processed by GoWrite
	Outbox() chan Message
}

// Pump is a struct that describes an entity with an inbox and
// and outbox channel of Messages, and their companion stop channels
// to let the pump know when to stop reading or writing
type Pump struct {
	InboxCh      chan Message
	ReadStopper  chan interface{}
	OutboxCh     chan Message
	WriteStopper chan interface{}
}

// Inbox returns the inbox channel of the pump
func (p *Pump) Inbox() chan Message {
	return p.InboxCh
}

// StopRead writes to the ReadStopper synchronization channel, thus indicating
// the pumper to stop reading
func (p *Pump) StopRead() {
	p.ReadStopper <- true
}

// Outbox returns the outbox channel of the pump
func (p *Pump) Outbox() chan Message {
	return p.OutboxCh
}

// StopWrite writes to the WriteStopper synchronization channel, thus indicating
// the pumper to stop writing
func (p *Pump) StopWrite() {
	p.WriteStopper <- true
}

// NewPump returns the address of a new value of the Pump struct with
// InboxCh and OutboxCh as buffered channels of size DefaultBufferSize
func NewPump() *Pump {
	return &Pump{
		InboxCh:      make(chan Message, DefaultBufferSize),
		ReadStopper:  make(chan interface{}),
		OutboxCh:     make(chan Message, DefaultBufferSize),
		WriteStopper: make(chan interface{}),
	}
}

// BidirectionalPumpConnection defines a connection between two
// pumpers such as the messages arriving at the inbox of one of them
// are relayed to the outbox of the other and vice-versa
type BidirectionalPumpConnection struct {
	Left  Pumper
	Right Pumper
	stop  chan interface{}
}

// NewBidirectionalPumpConnection returns the address of a new
// BidirectionalPumpConnection
func NewBidirectionalPumpConnection(left Pumper, right Pumper) *BidirectionalPumpConnection {
	return &BidirectionalPumpConnection{
		Left:  left,
		Right: right,
		stop:  make(chan interface{}),
	}
}

// Go spawns a goroutine for routing messages from each of the edges to the
// other
func (c BidirectionalPumpConnection) Go() {
	go func() {
		c.Left.GoRead()
		c.Left.GoWrite()
		c.Right.GoRead()
		c.Right.GoWrite()

		for {
			select {
			case m := <-c.Left.Inbox():
				log.Debugf("[%+v]: %s", c.Left, m)
				c.Right.Outbox() <- m
			case m := <-c.Right.Inbox():
				log.Debugf("[%+v]: %s", c.Right, m)
				c.Left.Outbox() <- m
			case <-c.stop:
				c.Left.StopRead()
				c.Left.StopWrite()
				c.Right.StopRead()
				c.Right.StopWrite()
				return
			}
		}
	}()

}

// Stop stops the goroutine started by Go
func (c BidirectionalPumpConnection) Stop() {
	c.stop <- true
}
