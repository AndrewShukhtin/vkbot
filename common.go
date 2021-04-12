package vkbot

import (
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

const (
	// VkAPIVersion current version od vk api
	VkAPIVersion = "5.130"
	// VkAPIUrl url to vk api
	VkAPIUrl     = "https://api.vk.com/method/"
)

var client = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

type internalError struct {
	Inner      error
	Message    string
	StackTrace string
	Misc       map[string]interface{}
}

func newInternalError(err error, messagef string, args ...interface{}) *internalError {
	return &internalError{
		Inner:      err,
		Message:    fmt.Sprintf(messagef, args...),
		StackTrace: string(debug.Stack()),
		Misc:       make(map[string]interface{}),
	}
}

func (err *internalError) Error() string {
	return fmt.Sprintf("message - %s; caused by - %s", err.Message, err.Inner)
}

func (err *internalError) SetMisc(name string, val interface{}) {
	err.Misc[name] = val
}