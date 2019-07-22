package cable

import (
	"fmt"
)

// Update is the interface of the values being interchanged by pumps
type Update interface{}

// Message is some contents sent by an author
type Message struct {
	Author   *Author
	Contents *Contents
}

func (m *Message) String() string {
	return fmt.Sprintf("%s: %s", m.Author, m.Contents)
}

// Author represents a platform independent author of a message
type Author struct {
	Name  string
	Alias string
}

func (a *Author) String() string {
	return fmt.Sprintf("%s", a.Alias)
}

// Contents represents a platform independent contents of a message
type Contents struct {
	Raw string
}

func (c *Contents) String() string {
	return c.Raw
}
