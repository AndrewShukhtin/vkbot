package event

import (
	"github.com/karlseguin/typed"
	"reflect"
	"testing"
)

func TestInvalidFields(t *testing.T) {
	malformedEvents := [...]map[string]interface{}{
		{},
		{"type": "test_event_type"},
		{"type": "test_event_type",
			"object": map[string]interface{}{
				"test_event_object": 0,
			}},
		{"type": "test_event_type",
			"object": map[string]interface{}{
				"test_event_object": 0,
			},
			"group_id": 0},
	}

	for _, me := range malformedEvents {
		_, err := NewEvent(me)
		if err == nil {
			t.Error("should be error while parsing event", me)
		}
	}
}

func TestSupportedEventsType(t *testing.T) {
	eventTypes := []string{
		MessageNewType,
		MessageReplyType,
		MessageEditType,
		MessageAllowType,
		MessageDenyType,
		MessageTypingStateType,
		MessageEventType,
	}
	for _, et := range eventTypes {
		e := map[string]interface{}{
			"type": et,
			"object": map[string]interface{}{
				"test_event_object": 0,
			},
			"group_id": 0,
			"event_id": "test_event_id"}
		_, err := NewEvent(e)
		if err != nil {
			t.Error("should not be error while parsing event", e)
		}
	}
}

func TestUnsupportedEventType(t *testing.T) {
	eventTypes := []string{
		"test_event_type",
	}
	for _, et := range eventTypes {
		e := map[string]interface{}{
			"type": et,
			"object": map[string]interface{}{
				"test_event_object": 0,
			},
			"group_id": 0,
			"event_id": "test_event_id"}
		_, err := NewEvent(e)
		if err == nil {
			t.Error("should be error while parsing event", e)
		}
	}
}

func TestEventMethods(t *testing.T) {
	rawEvents := []typed.Typed{
		{
			"type": "message_new",
			"object": typed.Typed{
				"test_obj": 22,
			},
			"group_id": 0,
			"event_id": "xxooxx",
		},
	}
	for _, re := range rawEvents {
		e, err := NewEvent(re)
		if err != nil {
			t.Error(err)
		}
		if e.GroupID() != re.Int("group_id") {
			t.Error("different group_ids")
		}
		if e.Type() != re.String("type") {
			t.Error("different types")
		}
		if e.EventID() != re.String("event_id") {
			t.Error("different event_ids")
		}
		if !reflect.DeepEqual(e.Object(), re.Object("object")) {
			t.Error("different objects")
		}
		if !reflect.DeepEqual(e.Data(), re) {
			t.Error("different events")
		}
	}
}
