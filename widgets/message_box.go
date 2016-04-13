package widgets

import (
	"fmt"
	"strings"

	"github.com/gizak/termui"
)

type Message struct {
	Content string
	Ticks   int

	ticksLeft int
}

type MessageBox struct {
	*termui.Par

	lhs []*Message
	rhs []*Message
}

func NewMessageBox() *MessageBox {
	par := termui.NewPar("")
	par.Border = false
	return &MessageBox{Par: par}
}

func tick(messages []*Message) ([]*Message, bool) {
	update := false
	keep := make([]*Message, 0, len(messages))
	for _, msg := range messages {
		if msg.ticksLeft > 0 {
			msg.ticksLeft--
		}
		if msg.ticksLeft != 0 {
			keep = append(keep, msg)
		} else {
			update = true
		}
	}
	return keep, update
}

func (m *MessageBox) Tick() bool {
	var updateL, updateR bool
	m.lhs, updateL = tick(m.lhs)
	m.rhs, updateR = tick(m.rhs)
	return updateL || updateR
}

type Side bool

const (
	Left  = Side(true)
	Right = Side(false)
)

func remove(msg *Message, list []*Message) []*Message {
	for i := range list {
		if list[i] == msg {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (m *MessageBox) addMessage(msg *Message, list, other []*Message) (
	newList, newOther []*Message) {
	newList, newOther = remove(msg, list), remove(msg, other)
	msg.ticksLeft = msg.Ticks
	newList = append([]*Message{msg}, newList...)
	return
}

func (m *MessageBox) AddMessage(msg *Message, side Side) {
	switch side {
	case Left:
		m.lhs, m.rhs = m.addMessage(msg, m.lhs, m.rhs)
	case Right:
		m.rhs, m.lhs = m.addMessage(msg, m.rhs, m.lhs)
	}
}

func join(msgs []*Message) string {
	s := make([]string, len(msgs))
	for i := range msgs {
		s[i] = msgs[i].Content
	}
	return strings.Join(s, ", ")
}

func (m *MessageBox) Buffer() termui.Buffer {
	lhs := join(m.lhs)
	rhs := join(m.rhs)
	space := m.InnerWidth() - len(clearStyles(rhs))

	m.Text = fmt.Sprintf(`%-*s%s`, space, lhs, rhs)

	return m.Par.Buffer()
}
