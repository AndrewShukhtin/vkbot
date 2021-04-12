package keyboard

import (
	"encoding/json"
)

type Keyboard struct {
	OneTime bool        `json:"one_time"`
	Buttons [][]*Button `json:"buttons"`
	Inline  bool        `json:"inline"`
}

type Button struct {
	Action Action `json:"action"`
	Color  string `json:"color,omitempty"`
}

func NewButton(act Action, clr string) *Button {
	return &Button{Action: act, Color: clr}
}

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

// Json get json representation of keyboard
func (k *Keyboard) Json() (string, error) {
	data, err := json.Marshal(k)
	return string(data), err
}
