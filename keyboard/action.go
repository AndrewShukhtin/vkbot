package keyboard

import (
	"encoding/json"
)

const (
	TextActionType     = "text"
	OpenLinkActionType = "open_link"
	LocationActionType = "location"
	VkPayActionType    = "vkpay"
	VkAppsActionType   = "open_app"
	CallbackActionType = "callback"
)

type Action interface {
	SetType(t string)
	GetType() string
	SetPayload(payload interface{})
	GetPayload() string
}

type BaseAction struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func (b *BaseAction) SetType(t string) {
	b.Type = t
}

func (b *BaseAction) GetType() string {
	return b.Type
}

func (b *BaseAction) SetPayload(payload interface{}) {
	data, err := json.Marshal(&payload)
	if err != nil {
		b.Payload = "{}"
		return
	}
	b.Payload = string(data)
}

func (b *BaseAction) GetPayload() string {
	return b.Payload
}

type TextAction struct {
	BaseAction
	Label   string `json:"label"`
}

func NewTextAction(label string) *TextAction {
	a := &TextAction{Label: label}
	a.SetType(TextActionType)
	a.Payload = "{}"
	return a
}

type OpenLinkAction struct {
	BaseAction
	Link    string `json:"link"`
	Label   string `json:"label"`
}

func NewOpenLinkAction(link string, label string) *OpenLinkAction {
	a := &OpenLinkAction{Link: link, Label: label}
	a.SetType(OpenLinkActionType)
	a.Payload = "{}"
	return a
}

type LocationAction struct {
	BaseAction
}

func NewLocationAction() *LocationAction {
	a := &LocationAction{}
	a.SetType(LocationActionType)
	a.Payload = "{}"
	return a
}

type VkPayAction struct {
	BaseAction
	Hash    string `json:"hash"`
}

func NewVkPayAction(hash string) *VkPayAction {
	a := &VkPayAction{}
	a.SetType(VkPayActionType)
	a.Payload = "{}"
	return a
}

type VkAppsAction struct {
	BaseAction
	AppId   int    `json:"app_id"`
	OwnerId int    `json:"owner_id"`
	Label   string `json:"label"`
	Hash    string `json:"hash"`
}

func NewVkAppsAction(label string) *VkAppsAction {
	a := &VkAppsAction{Label: label}
	a.SetType(VkAppsActionType)
	a.Payload = "{}"
	return a
}

type CallbackAction struct {
	BaseAction
	Label   string `json:"label"`
}

func NewCallbackAction(label string) *CallbackAction {
	a := &CallbackAction{Label: label}
	a.SetType(CallbackActionType)
	a.Payload = "{}"
	return a
}