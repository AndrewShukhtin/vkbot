package event

import (
	"fmt"
	"github.com/karlseguin/typed"
)

type (
	// Event interface for event data
	Event interface {
		// Data returns all event data
		Data() typed.Typed

		// Type returns event type
		Type() string

		// Object returns event object
		Object() typed.Typed

		// GroupID returns event object
		GroupID() int

		// EventID returns event event_id
		EventID() string
	}

	event struct {
		data typed.Typed
	}
)

// NewEvent parse date to event data
func NewEvent(data map[string]interface{}) (Event, error) {
	return parseToEventType(data)
}

func (e *event) Data() typed.Typed {
	return e.data
}

func (e *event) Type() string {
	return e.data.String("type")
}

func (e *event) Object() typed.Typed {
	return e.data.Object("object")
}

func (e *event) GroupID() int {
	return e.data.Int("group_id")
}

func (e *event) EventID() string {
	return e.data.String("event_id")
}

func parseToEventType(update typed.Typed) (*event, error) {
	if _, ok := update.StringIf("type"); !ok {
		return nil, fmt.Errorf("event invalid 'type' field")
	}
	if _, ok := update.ObjectIf("object"); !ok {
		return nil, fmt.Errorf("event invalid 'object' field")
	}
	if _, ok := update.IntIf("group_id"); !ok {
		return nil, fmt.Errorf("event invalid 'group_id' field")
	}
	if _, ok := update.StringIf("event_id"); !ok {
		return nil, fmt.Errorf("event invalid 'event_id' field")
	}

	switch update.String("type") {
	case MessageNewType:
	case MessageReplyType, MessageEditType:
	case MessageAllowType:
	case MessageDenyType:
	case MessageTypingStateType:
	case MessageEventType:
	default:
		return nil, fmt.Errorf("not supported event type")
	}
	return &event{data: update}, nil
}
