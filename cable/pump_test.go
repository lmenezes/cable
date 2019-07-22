package cable

import (
	. "github.com/stretchr/testify/assert"
	"testing"
)

/* fake Pumper */

type fakePumper struct {
	*Pump
}

func newFakePumper() *fakePumper {
	fp := &fakePumper{}
	fp.Pump = NewPump(fp)
	return fp
}

func (*fakePumper) GoRead()                                      {}
func (*fakePumper) GoWrite()                                     {}
func (*fakePumper) NextEvent() Update                            { panic("Not implemented") }
func (*fakePumper) ToInboxUpdate(interface{}) (Update, error)    { panic("Not implemented") }
func (*fakePumper) FromOutboxUpdate(Update) (interface{}, error) { panic("Not implemented") }
func (*fakePumper) Send(update interface{}) error                { panic("Not implemented") }

/* fake Update */

type fakeMessage struct {
	text string
}

func (fm *fakeMessage) Value() *Message {
	return &Message{
		Contents: &Contents{fm.text},
		Author:   &Author{Name: "Anonymous", Alias: "@acoward"},
	}
}

func TestBidirectionalPumpConnection(t *testing.T) {
	left := newFakePumper()
	right := newFakePumper()

	bidi := NewBidirectionalPumpConnection(left, right)
	bidi.Go()

	left.Inbox() <- &fakeMessage{text: "Fed into left"}
	right.Inbox() <- &fakeMessage{text: "Fed into right"}

	Equal(t, "Fed into left", (<-right.Outbox()).(*fakeMessage).text)
	Equal(t, "Fed into right", (<-left.Outbox()).(*fakeMessage).text)

	bidi.Stop()
}
