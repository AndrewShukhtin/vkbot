package vkbot

import (
	"context"
	"fmt"
	"github.com/karlseguin/typed"
	"golang.org/x/time/rate"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGroupLongPollServer_Settings(t *testing.T) {
	in := Params{"message_event": 1, "test_event": 1}
	s := NewGroupLongPollServer(nil, 0)
	s.SetSettings(in)

	out := s.Settings()
	if _, ok := out["message_event"]; !ok {
		t.Error("should contain 'message_event'")
	}
	if i := out["message_event"].(int); i != 1 {
		t.Error("'message_event' should be equal 1")
	}
	if _, ok := out["test_event"]; ok {
		t.Error("should not contain 'test_event'")
	}
}

type fakeVkAPI struct {
	respByMethod map[string]typed.Typed
}

func newFakeVkAPI(respByMethod map[string]typed.Typed) *fakeVkAPI {
	return &fakeVkAPI{respByMethod: respByMethod}
}

func (api *fakeVkAPI) CallMethod(methodName string, params Params) (typed.Typed, error) {
	if resp, ok := api.respByMethod[methodName]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("method not found")
}

func TestGroupLongPollServerFailedInit(t *testing.T) {
	tests := []map[string]typed.Typed{
		{},
		{"groups.setLongPollSettings": {}},
	}
	for _, test := range tests {
		s := NewGroupLongPollServer(newFakeVkAPI(test), 0)
		err := s.Init()
		if err == nil {
			t.Errorf("should be error while initializing groupLongPollServer")
		}
	}
}

func TestGroupLongPollServerInit(t *testing.T) {
	testResp := map[string]typed.Typed{
		"groups.setLongPollSettings": {},
		"groups.getLongPollServer": {
			"ts":     "test_ts",
			"key":    "test_key",
			"server": "test_server",
		},
	}
	s := &groupLongPollServer{VkAPI: newFakeVkAPI(testResp), GroupID: 0, mtx: &sync.Mutex{}}
	err := s.Init()
	if err != nil {
		t.Errorf("should not be error while initializing groupLongPollServer")
	}
	if s.Ts != "test_ts" || s.Key != "test_key" || s.Server != "test_server" {
		t.Errorf("invalid initialization of groupLongPollServer fields")
	}
}

func TestGroupLongPollServer_getUpdate(t *testing.T) {
	type TestCase struct {
		Name    string // test case Name
		VkAPI   VkAPI
		Server  *httptest.Server
		URL     string
		Context context.Context
	}

	servers := map[string]*httptest.Server{
		"simple": httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "max_attempts") {
				w.WriteHeader(http.StatusInternalServerError)
			}
			if strings.Contains(r.URL.Path, "failed") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"fail": 1}`))
			}
			if strings.Contains(r.URL.Path, "fine") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"type": "test_event", "object" : {"test": 0}, "group_id": 0, "event_id": "xxooxx"}`))
			}
		})),
	}
	testCases := []TestCase{
		{
			Name:   "failed request creation case",
			Server: servers["simple"],
		},
		{
			Name:    "failed while making request",
			Server:  servers["simple"],
			Context: context.Background(),
			URL:     "invalid_url",
		},
		{
			Name:    "failed convert body to json",
			Server:  servers["simple"],
			Context: context.Background(),
			URL:     servers["simple"].URL,
		},
		{
			Name:    "exceed max number of attempts",
			Server:  servers["simple"],
			Context: context.Background(),
			URL:     servers["simple"].URL + "/max_attempts",
		},
		{
			Name:    "groupLongPollServer returned failed and reinitialization failed",
			VkAPI:   newFakeVkAPI(map[string]typed.Typed{}),
			Server:  servers["simple"],
			Context: context.Background(),
			URL:     servers["simple"].URL + "/failed",
		},
		{
			Name: "groupLongPollServer returned failed, reinitialization go well, but server always return 'fail'",
			VkAPI: newFakeVkAPI(map[string]typed.Typed{"groups.getLongPollServer": {
				"ts":     "test_ts",
				"key":    "test_key",
				"server": servers["simple"].URL + "/failed",
			}}),
			Server:  servers["simple"],
			Context: context.Background(),
			URL:     servers["simple"].URL + "/failed",
		},
	}
	s := &groupLongPollServer{mtx: &sync.Mutex{}}
	for _, tc := range testCases {
		s.VkAPI = tc.VkAPI
		s.Server = tc.URL
		s.client = tc.Server.Client()
		s.eventCtx = tc.Context

		out := s.getUpdate()
		respAndErr := <-out
		if respAndErr.UnpackedResponse != nil && respAndErr.Error == nil {
			t.Errorf("should be error and nil response")
		}
		fmt.Println(respAndErr.Error)
	}

	fineTestCase := TestCase{
		Name:    "groupLongPollServer returned wel-formed response (api event)",
		Server:  servers["simple"],
		Context: context.Background(),
		URL:     servers["simple"].URL + "/fine",
	}
	s.VkAPI = fineTestCase.VkAPI
	s.Server = fineTestCase.URL
	s.client = fineTestCase.Server.Client()
	s.eventCtx = fineTestCase.Context

	out := s.getUpdate()
	respAndErr := <-out
	if respAndErr.UnpackedResponse == nil && respAndErr.Error != nil {
		t.Errorf("should not be error and no nil response: it's fine test")
	}

	for _, server := range servers {
		server.Close()
	}
}

func TestGroupLongPollServer_AtOverheat(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		}))

	done := make(chan bool, 1)
	defer close(done)
	d := &hookDealer{
		AtOverheat: func(context.Context) bool {
			done <- true
			return true
		},
		AtLimit: func(_ context.Context, _ *rate.Limiter) {

		},
		AtResponseError: func(o *overHeater, err error) {
			o.addTimeStamp(time.Now())
		},
		AtNewUpdateError: func(_ *overHeater, _ error) {

		},
	}

	s := groupLongPollServer{
		config:     LongPollConfig{Limiter: rate.NewLimiter(rate.Inf, 0)},
		client:     server.Client(),
		hookDealer: d,
	}
	s.eventCtx, s.eventCancel = context.WithCancel(context.Background())

	timer := time.NewTimer(time.Millisecond * 100)

	s.StartUpdatesLoop()
	select {
	case val := <-done:
		if val != true {
			t.Error("channel should pass true when too many errors would occurred")
		}
	case <-timer.C:
		t.Error("timed out")
	}
	s.eventCancel()
}

func TestGroupLongPollServer_AtLimit(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		}))

	d := &hookDealer{
		AtOverheat: func(context.Context) bool {
			return false
		},
		AtResponseError: func(_ *overHeater, _ error) {

		},
		AtNewUpdateError: func(_ *overHeater, _ error) {

		},
	}

	s := groupLongPollServer{
		config:     LongPollConfig{Limiter: rate.NewLimiter(1, 0)},
		client:     server.Client(),
		hookDealer: d,
	}
	s.eventCtx, s.eventCancel = context.WithCancel(context.Background())

	timer := time.NewTimer(time.Millisecond * 100)

	done := make(chan bool)
	defer close(done)
	d.AtLimit = func(_ context.Context, _ *rate.Limiter) {
		done <- true
		s.eventCancel()
	}

	s.StartUpdatesLoop()

	select {
	case val := <-done:
		if val != true {
			t.Error("channel should pass true when rate limiter in use")
		}
	case <-timer.C:
		t.Error("timed out")
	}
}

func TestGroupLongPollServer_AtResponseError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		}))

	done := make(chan bool, 1)
	defer close(done)
	d := &hookDealer{
		AtOverheat: func(context.Context) bool {
			return false
		},
		AtLimit: func(_ context.Context, _ *rate.Limiter) {

		},
		AtNewUpdateError: func(_ *overHeater, _ error) {

		},
	}

	s := groupLongPollServer{
		config:     LongPollConfig{Limiter: rate.NewLimiter(rate.Inf, 0)},
		client:     server.Client(),
		hookDealer: d,
	}
	s.eventCtx, s.eventCancel = context.WithCancel(context.Background())

	timer := time.NewTimer(time.Millisecond * 100)

	d.AtResponseError = func(_ *overHeater, _ error) {
		done <- true
		s.eventCancel()
	}

	s.StartUpdatesLoop()
	select {
	case val := <-done:
		if val != true {
			t.Error("channel should pass true when rate limiter in use")
		}
	case <-timer.C:
		t.Error("timed out")
	}
	s.eventCancel()
}

func TestGroupLongPollServer_AtNewUpdateError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ts": 0, "updates" : [{"type": "test_event", "object" : {"test": 0}, "group_id": 0, "event_id": "xxooxx"}]}`))
		}))

	done := make(chan bool, 1)
	defer close(done)
	d := &hookDealer{
		AtOverheat: func(_ context.Context) bool {
			return false
		},
		AtLimit: func(_ context.Context, _ *rate.Limiter) {

		},
		AtResponseError: func(_ *overHeater, _ error) {

		},
	}

	s := groupLongPollServer{
		config:     LongPollConfig{Limiter: rate.NewLimiter(rate.Inf, 0)},
		client:     server.Client(),
		Server:     server.URL,
		mtx:        &sync.Mutex{},
		hookDealer: d,
		eventCtx:   context.Background(),
	}

	timer := time.NewTimer(time.Millisecond * 100)

	d.AtNewUpdateError = func(_ *overHeater, _ error) {
		done <- true
		s.eventCancel()
	}

	s.StartUpdatesLoop()
	select {
	case val := <-done:
		if val != true {
			t.Error("channel should pass true when rate limiter in use")
		}
	case <-timer.C:
		t.Error("timed out")
	}
	s.eventCancel()
}

func TestParseToUpdate(t *testing.T) {
	testCases := []typed.Typed{
		{
			"ts": 1,
		},
		{
			"ts":      1,
			"updates": "not an array",
		},
		{
			"ts":      1,
			"updates": []typed.Typed{},
		},
		{
			"ts": 1,
			"updates": []typed.Typed{
				{
					"type": "test_type",
				},
			},
		},
		{
			"ts": 1,
			"updates": []typed.Typed{
				{
					"type":     "message_new",
					"object":   typed.Typed{},
					"group_id": 0,
					"event_id": "test_event_id",
				},
			},
		},
	}
	for i, tc := range testCases {
		if i == len(testCases)-1 {
			_, err := NewUpdate(tc)
			if err != nil {
				t.Error("should not be error")
			}
		} else {
			_, err := NewUpdate(tc)
			if err == nil {
				t.Error("should be error")
			}
		}
	}
}

func TestGroupLongPollServer_StopUpdatesLoop_BeforeStart(t *testing.T) {
	s := &groupLongPollServer{}
	defer func() {
		if msg := recover().(string); msg != "trying to stop not started event loop" {
			t.Error("no recover")
		}
	}()
	s.StopUpdatesLoop()
}

func TestGroupLongPollServer_StopUpdatesLoop_AfterInit(t *testing.T) {
	s := &groupLongPollServer{}
	s.eventCtx, s.eventCancel = context.WithCancel(context.Background())
	s.StopUpdatesLoop()
}

func TestGroupLongPollServer_SetConfig(t *testing.T) {
	s := &groupLongPollServer{}
	cfg := LongPollConfig{
		Wait:             0,
		Limiter:          nil,
		UpdateBufferSize: -1,
	}
	s.SetConfig(cfg)
	if s.config.Wait != 25 || s.config.Limiter == nil || s.config.UpdateBufferSize != 10 {
		t.Error("invalid settings")
	}
}

func TestNewUpdate(t *testing.T) {
	testCases := []typed.Typed{
		{
			"ts": "1",
			"updates": []typed.Typed{
				{
					"type":     "message_new",
					"object":   typed.Typed{},
					"group_id": 0,
					"event_id": "test_event_id",
				},
			},
		},
	}
LOOP:
	for _, tc := range testCases {
		u, err := NewUpdate(tc)
		if err != nil {
			t.Error("error occurred", err)
			continue
		}
		if u.Ts() != tc.String("ts") {
			t.Error("wrong ts")
			continue
		}
		tce := tc.Objects("updates")
		if len(tce) != len(u.Events()) {
			t.Error("wrong length")
			continue
		}
		for i, e := range u.Events() {
			if !reflect.DeepEqual(e.Data(), tce[i]) {
				t.Error("different objects")
				continue LOOP
			}
		}
	}
}
