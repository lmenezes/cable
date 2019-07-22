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
	return &fakePumper{NewPump()}
}

func (*fakePumper) GoRead() {}

func (*fakePumper) GoWrite() {}

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
