package keyboard

import (
	"encoding/json"
)

// Various types of action supported by vk api
const (
	TextActionType     = "text"
	OpenLinkActionType = "open_link"
	LocationActionType = "location"
	VkPayActionType    = "vkpay"
	VkAppsActionType   = "open_app"
	CallbackActionType = "callback"
)

// Action basic interface
type Action interface {
	// SetType set type of action
	SetType(t string)

	// GetType get type of action
	GetType() string

	// SetPayload set payload of action
	SetPayload(payload interface{})

	// GetPayload get payload of action
	GetPayload() string
}

// BaseAction base action fields
type BaseAction struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

// SetType set type of action
func (b *BaseAction) SetType(t string) {
	b.Type = t
}

// GetType get type of action
func (b *BaseAction) GetType() string {
	return b.Type
}

// SetPayload set payload of action
func (b *BaseAction) SetPayload(payload interface{}) {
	data, err := json.Marshal(&payload)
	if err != nil {
		b.Payload = "{}"
		return
	}
	b.Payload = string(data)
}

// GetPayload get payload of action
func (b *BaseAction) GetPayload() string {
	return b.Payload
}

// TextAction action of text type
type TextAction struct {
	BaseAction
	Label   string `json:"label"`
}

// NewTextAction new action of text type
func NewTextAction(label string) *TextAction {
	a := &TextAction{Label: label}
	a.SetType(TextActionType)
	a.Payload = "{}"
	return a
}

// OpenLinkAction action of open_link type
type OpenLinkAction struct {
	BaseAction
	Link    string `json:"link"`
	Label   string `json:"label"`
}

// NewOpenLinkAction new action of open_link type
func NewOpenLinkAction(link string, label string) *OpenLinkAction {
	a := &OpenLinkAction{Link: link, Label: label}
	a.SetType(OpenLinkActionType)
	a.Payload = "{}"
	return a
}

// LocationAction action of location type
type LocationAction struct {
	BaseAction
}

// NewLocationAction new action of location type
func NewLocationAction() *LocationAction {
	a := &LocationAction{}
	a.SetType(LocationActionType)
	a.Payload = "{}"
	return a
}

// VkPayAction action of vkpay type
type VkPayAction struct {
	BaseAction
	Hash    string `json:"hash"`
}

// NewVkPayAction new  action of vkpay type
func NewVkPayAction(hash string) *VkPayAction {
	a := &VkPayAction{}
	a.SetType(VkPayActionType)
	a.Payload = "{}"
	return a
}

// VkAppsAction action of open_app type
type VkAppsAction struct {
	BaseAction
	AppID   int    `json:"app_id"`
	OwnerID int    `json:"owner_id"`
	Label   string `json:"label"`
	Hash    string `json:"hash"`
}

// NewVkAppsAction new action of open_app type
func NewVkAppsAction(label string) *VkAppsAction {
	a := &VkAppsAction{Label: label}
	a.SetType(VkAppsActionType)
	a.Payload = "{}"
	return a
}

// CallbackAction action of callback type
type CallbackAction struct {
	BaseAction
	Label   string `json:"label"`
}

// NewCallbackAction new action of callback type
func NewCallbackAction(label string) *CallbackAction {
	a := &CallbackAction{Label: label}
	a.SetType(CallbackActionType)
	a.Payload = "{}"
	return a
}