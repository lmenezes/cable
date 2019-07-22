package cable

import (
	. "github.com/stretchr/testify/assert"
	"testing"
)

func TestBidirectionalPumpConnection(t *testing.T) {
	left := newFakePumper()
	right := newFakePumper()

	bidi := NewBidirectionalPumpConnection(left, right)
	bidi.Go()
	defer bidi.Stop()

	left.Inbox() <- &fakeMessage{text: "Fed into left"}
	right.Inbox() <- &fakeMessage{text: "Fed into right"}

	Equal(t, "Fed into left", (<-right.Outbox()).String())
	Equal(t, "Fed into right", (<-left.Outbox()).String())
}
