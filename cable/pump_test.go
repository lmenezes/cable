package cable

import (
	. "github.com/stretchr/testify/assert"
	"testing"
)

func TestBidirectionalPumpConnection(t *testing.T) {
	left := NewPump()
	right := NewPump()

	bidi := NewBidirectionalPumpConnection(left, right)
	bidi.Go()

	left.Inbox <- &fakeMessage{text: "Fed into left"}
	right.Inbox <- &fakeMessage{text: "Fed into right"}

	bidi.Stop()

	Equal(t, "Fed into left", (<-right.Outbox).String())
	Equal(t, "Fed into right", (<-left.Outbox).String())
}
