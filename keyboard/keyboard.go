package keyboard

import (
	"encoding/json"
)

// Keyboard allows yo construct keyboard of bot
type Keyboard struct {
	OneTime bool        `json:"one_time"`
	Buttons [][]*Button `json:"buttons"`
	Inline  bool        `json:"inline"`
}

// Button on bot's keyboard
type Button struct {
	Action Action `json:"action"`
	Color  string `json:"color,omitempty"`
}

// NewButton new button with action and color
func NewButton(act Action, clr string) *Button {
	return &Button{Action: act, Color: clr}
}

// NewKeyboard new keyboard
func NewKeyboard(oneTime bool, inline bool) *Keyboard {
	return &Keyboard{
		OneTime:       oneTime,
		Buttons:       make([][]*Button, 0),
		Inline:        inline,
	}
}

// AddButton adds one button to the full width of the keyboard
func (k *Keyboard) AddButton(button *Button) {
	sb := []*Button{button}
	k.Buttons = append(k.Buttons, sb)
}

// AddButtons adds buttons on one row of keyboard
func (k *Keyboard) AddButtons(buttons []*Button) {
	k.Buttons = append(k.Buttons, buttons)
}

// JSON get json representation of keyboard
func (k *Keyboard) JSON() (string, error) {
	data, err := json.Marshal(k)
	return string(data), err
}
