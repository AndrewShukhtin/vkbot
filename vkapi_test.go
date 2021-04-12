package vkbot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestInvalidPostForm(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	api := vkApi{
		URL: server.URL,
		client: server.Client(),
	}
	defer server.Close()

	_, err := api.CallMethod("test.Tests", Params{"test": "test"})
	if err == nil {
		t.Error("should be error while making request")
	}
}

func TestInvalidJsonResponse(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	api := vkApi{
		URL: server.URL + "/",
		client: server.Client(),
	}
	defer server.Close()

	_, err := api.CallMethod("test.Tests", Params{"test" : "test"})
	if err == nil {
		t.Error("should be error while making request")
	}
}

func TestVkApiErrorResponse(t *testing.T) {
	vkApiErrorResponse :=[]byte(
		`{"error": "dumb_error"}`,
		)
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(vkApiErrorResponse)
		}))

	api := vkApi{
		URL: server.URL + "/",
		client: server.Client(),
	}
	defer server.Close()

	_, err := api.CallMethod("test.Tests", Params{"test" : "test"})
	if err == nil {
		t.Error("should be error while making request")
	}
}

func TestVkApiNoResponseField(t *testing.T) {
	vkApiResponse :=[]byte(
		`{"field": "dump_field"}`,
	)
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(vkApiResponse)
		}))

	api := vkApi{
		URL: server.URL + "/",
		client: server.Client(),
	}
	defer server.Close()

	_, err := api.CallMethod("test.Tests", Params{"test" : "test"})
	if err == nil {
		t.Error("should be error while making request")
	}
}

func TestVkApiNotJsonObjectResponseField(t *testing.T) {
	vkApiResponse :=[]byte(
		`{"response": "dump_field"}`,
	)
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(vkApiResponse)
		}))

	api := vkApi{
		URL: server.URL + "/",
		client: server.Client(),
	}
	defer server.Close()

	_, err := api.CallMethod("test.Tests", Params{"test" : "test"})
	if err == nil {
		t.Error("should be error while making request")
	}
}

func TestVkApiNumberResponse(t *testing.T) {
	vkApiResponse :=[]byte(
		`{"response": 1}`,
	)
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(vkApiResponse)
		}))

	api := vkApi{
		URL: server.URL + "/",
		client: server.Client(),
	}
	defer server.Close()

	resp, err := api.CallMethod("test.Tests", Params{"test" : "test"})
	if err != nil {
		t.Error("should not be error while making request")
	}

	var i interface{}
	json.Unmarshal(vkApiResponse, &i)

	if reflect.DeepEqual(resp, i) {
		t.Errorf("resp: %v\n vkApiResponse: %v", resp, i)
	}
}

func TestVkApiResponse(t *testing.T) {
	vkApiResponse :=[]byte(
		`{"response": {"vk_object": "test"}}`,
	)
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(vkApiResponse)
		}))

	api := vkApi{
		URL: server.URL + "/",
		client: server.Client(),
	}
	defer server.Close()

	resp, err := api.CallMethod("test.Tests", Params{"test" : "test"})
	if err != nil {
		t.Error("should not be error while making request")
	}

	var i interface{}
	json.Unmarshal(vkApiResponse, &i)

	if reflect.DeepEqual(resp, i) {
		t.Errorf("resp: %v\n vkApiResponse: %v", resp, i)
	}
}
