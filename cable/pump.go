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
	// TODO: rename GoRead -> ReadPump
	GoRead()
	// StopRead stops the read goroutine
	StopRead()
	// Inbox returns a channel containing the messages read by the ReadPumper
	Inbox() chan Update
	// NextEvent returns the next event to read from the read pump
	// TODO: rename NextUpdate -> Read
	NextEvent() Update
	// ToInboxUpdate knows how to convert anything being read to an Update
	ToInboxUpdate(interface{}) (Update, error)
}

// WritePumper is the interface implemented by Write pumpers.
// Write pumpers read events fed into the outbox and write them somewhere else
type WritePumper interface {
	// GoRead spawns a new goroutine to write messages arriving at the outbox
	// TODO: rename GoWrite -> WritePump
	GoWrite()
	// StopWrite stops the write goroutine
	StopWrite()
	// Outbox returns a channel of messages, which will be processed by GoWrite
	Outbox() chan Update
	// FromOutboxUpdate knows how to convert anything arriving at the outbox, to
	// something that can be sent.
	FromOutboxUpdate(Update) (interface{}, error)
	// Send sends the converted update arriving at the outbox
	// TODO: rename Send -> Write
	Send(update interface{}) error
}

// Pump is a struct that describes an entity with an inbox and
// and outbox channel of Messages, and their companion stop channels
// to let the pump know when to stop reading or writing
type Pump struct {
	InboxCh      chan Update
	ReadStopper  chan interface{}
	OutboxCh     chan Update
	WriteStopper chan interface{}
	pumper       Pumper
}

// GoRead makes telegram listen for messages in a different goroutine.
// Those messages will be pushed to the InboxCh of the Pump.
//
// The goroutine can be stopped by feeding ReadStopper synchronization channel
// which can be done by calling StopRead()
func (p *Pump) GoRead() {
	go func() {
		for {
			select {
			case <-p.ReadStopper:
				return
			default:
				ev := p.pumper.NextEvent()
				update, err := p.pumper.ToInboxUpdate(ev)
				if err != nil {
					log.Debugf("Update from inbox discarded: %s", err)
				} else {
					p.Inbox() <- update
				}
			}
		}
	}()
}

// StopRead writes to the ReadStopper synchronization channel, thus indicating
// the pumper to stop reading
func (p *Pump) StopRead() {
	p.ReadStopper <- true
}

// Inbox returns the inbox channel of the pump
func (p *Pump) Inbox() chan Update {
	return p.InboxCh
}

// GoWrite spawns a goroutine that takes care of delivering to telegram the
// messages arriving at the OutboxCh of the Pump.
//
// The goroutine can be stopped by feeding WriteStopper synchronization channel
// which can be done by calling StopWrite()
func (p *Pump) GoWrite() {
	go func() {
		for {
			select {
			case ou := <-p.Outbox():
				update, err := p.pumper.FromOutboxUpdate(ou)
				if err != nil {
					log.Debugf("Update from inbox discarded: %s", err)
				}
				err = p.pumper.Send(update)
				if err != nil {
					log.Errorln("Error sending message: ", err)
				}
			case <-p.WriteStopper:
				return
			}
		}
	}()
}

// StopWrite writes to the WriteStopper synchronization channel, thus indicating
// the pumper to stop writing
func (p *Pump) StopWrite() {
	p.WriteStopper <- true
}

// Outbox returns the outbox channel of the pump
func (p *Pump) Outbox() chan Update {
	return p.OutboxCh
}

// NewPump returns the address of a new value of the Pump struct with
// InboxCh and OutboxCh as buffered channels of size DefaultBufferSize
func NewPump(pumper Pumper) *Pump {
	return &Pump{
		InboxCh:      make(chan Update, DefaultBufferSize),
		ReadStopper:  make(chan interface{}),
		OutboxCh:     make(chan Update, DefaultBufferSize),
		WriteStopper: make(chan interface{}),
		pumper:       pumper,
	}
}

/* Bidirectional pump connection */

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
