package cable

import (
	"fmt"
	telegramAPI "github.com/go-telegram-bot-api/telegram-bot-api"
	slackAPI "github.com/nlopes/slack"
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

/* fake Message */

type fakeMessage struct {
	text string
}

func (fm fakeMessage) ToSlack() ([]slackAPI.MsgOption, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (fm fakeMessage) ToTelegram(telegramChatID int64) (telegramAPI.MessageConfig, error) {
	return telegramAPI.MessageConfig{}, fmt.Errorf("Not implemented")
}

func (fm fakeMessage) String() string {
	return fm.text
}

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
