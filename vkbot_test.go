package vkbot

import (
	"fmt"
	"github.com/AndrewShukhtin/vkbot/event"
	"github.com/karlseguin/typed"
	"sync"
	"testing"
	"time"
)

func TestDispatcherAndWorkersHandleAllIncomingEvents(t *testing.T) {
	d := newDispatcher(2, 0)
	wg := &sync.WaitGroup{}
	wg.Add(5)
	d.setWorkerFunc(func(_ event.Event) { wg.Done() })
	eventChan := make(chan event.Event, 2)
	go func() {
		for i := 0; i < 5; i++ {
			eventChan <- nil
		}
	}()
	d.startWorkers()
	go d.dispatch(eventChan)
	wg.Wait()
	d.stopWorkers(func() {})
	close(eventChan)
}

func TestDispatcherCancellation(t *testing.T) {
	d := newDispatcher(2, 0)
	d.setWorkerFunc(func(_ event.Event) {})
	eventChan := make(chan event.Event, 0)
	d.startWorkers()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go d.dispatch(eventChan)
	d.stopWorkers(func() {
		wg.Done()
	})
	wg.Wait()
	close(eventChan)
}

func TestVkBot_SetConfig(t *testing.T) {
	bot := &VkBot{}
	cfg := BotConfig{Workers: 10, WorkerBuffer: 10, Events: 10}
	bot.SetConfig(cfg)

	if bot.config != cfg {
		t.Error("configs should be equal")
	}
}

func TestVkBot_Init(t *testing.T) {
	type HandlerInfo struct {
		EventType  string
		HandleFunc HandleFunc
	}
	type TestCase struct {
		Name                string
		VkAPI               VkAPI
		GroupLongPollServer GroupLongPollServer
		HandlerInfo         HandlerInfo
		ShouldBeError       bool
	}
	testCases := []TestCase{
		{
			Name:                "not existing event",
			HandlerInfo:         HandlerInfo{EventType: "test_event", HandleFunc: nil},
			ShouldBeError:       true,
			GroupLongPollServer: &groupLongPollServer{settings: Params{"fake_event": 1}},
		},
		{
			Name:                "nil handler",
			HandlerInfo:         HandlerInfo{EventType: "fake_event", HandleFunc: nil},
			ShouldBeError:       true,
			GroupLongPollServer: &groupLongPollServer{settings: Params{"fake_event": 1}},
		},
		{
			Name:          "vk api error",
			HandlerInfo:   HandlerInfo{EventType: "fake_event", HandleFunc: notFoundHandler},
			ShouldBeError: true,
			GroupLongPollServer: &groupLongPollServer{
				settings: Params{"fake_event": 1},
				VkAPI:    newFakeVkAPI(map[string]typed.Typed{}),
			},
		},
		{
			Name:          "vk api error",
			HandlerInfo:   HandlerInfo{EventType: "fake_event", HandleFunc: notFoundHandler},
			ShouldBeError: false,
			GroupLongPollServer: &groupLongPollServer{
				settings: Params{"fake_event": 1},
				VkAPI: newFakeVkAPI(map[string]typed.Typed{
					"groups.setLongPollSettings": {},
					"groups.getLongPollServer": {
						"ts":     "test_ts",
						"key":    "test_key",
						"server": "test_server",
					},
				}),
				mtx: &sync.Mutex{},
			},
		},
	}

	for _, tc := range testCases {
		bot := &VkBot{
			vkAPI:          tc.VkAPI,
			handlers:       map[string]HandleFunc{},
			longPollServer: tc.GroupLongPollServer,
		}
		bot.EventHandler(tc.HandlerInfo.EventType, tc.HandlerInfo.HandleFunc)
		err := bot.Init()
		if tc.ShouldBeError {
			if err == nil {
				t.Error("should be error")
			}
		} else {
			if err != nil {
				t.Error("should not be error")
			}
		}
	}
}

func TestVkBot_handleEvent(t *testing.T) {
	type TestCase struct {
		Name string
		withError  bool
	}

	testCases := []TestCase {
		{
			Name: "without error",
			withError: false,
		},
		{
			Name: "with error",
			withError: true,
		},
	}

	for _, tc := range testCases {
		done := make(chan bool)
		defer close(done)

		bot := &VkBot{handlers: map[string]HandleFunc{}}
		if tc.withError {
			bot.EventHandler(event.MessageNewType, func(_ event.Event) error {
				done <- true
				return fmt.Errorf("test error")
			})
		} else {
			bot.EventHandler(event.MessageNewType, func(_ event.Event) error {
				done <- true
				return nil
			})
		}

		e, _ := event.NewEvent(typed.Typed{
			"type": event.MessageNewType,
			"object": map[string]interface{}{
				"test_event_object": 0,
			},
			"group_id": 0,
			"event_id": "test_event_id",
		})
		go bot.handleEvent(e)
		<-done
	}
}

type fakeLongPollServer struct {
	hooksByMethods       map[string]hookFunc
	startUpdatesLoopFunc func() <-chan Update
}

func newFakeLongPollServer() *fakeLongPollServer {
	return &fakeLongPollServer{
		hooksByMethods: map[string]hookFunc{},
	}
}

func (f *fakeLongPollServer) Settings() Params {
	if hf, ok := f.hooksByMethods["Settings"]; ok {
		hf()
	}
	return nil
}

func (f *fakeLongPollServer) SetSettings(_ Params) {
	if hf, ok := f.hooksByMethods["SetSettings"]; ok {
		hf()
	}
}

func (f *fakeLongPollServer) SetConfig(_ LongPollConfig) {
	if hf, ok := f.hooksByMethods["SetConfig"]; ok {
		hf()
	}
}

func (f *fakeLongPollServer) Init() error {
	if hf, ok := f.hooksByMethods["Init"]; ok {
		hf()
	}
	return nil
}

func (f *fakeLongPollServer) StartUpdatesLoop() <-chan Update {
	return f.startUpdatesLoopFunc()
}

func (f *fakeLongPollServer) StopUpdatesLoop() {
	if hf, ok := f.hooksByMethods["StopUpdatesLoop"]; ok {
		hf()
	}
}

func TestVkBot_Stop(t *testing.T) {
	longPollServer := newFakeLongPollServer()
	bot := VkBot{
		longPollServer: longPollServer,
		dispatcher:     newDispatcher(2, 2),
	}
	done := make(chan bool)
	defer close(done)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	longPollServer.hooksByMethods["StopUpdatesLoop"] = func() {
		go func() {
			done <- true
			wg.Done()
		}()
	}

	eventChan := make(chan event.Event)
	defer close(eventChan)

	bot.dispatcher.workerFunc = func(_ event.Event) {}
	bot.dispatcher.startWorkers()
	go bot.dispatcher.dispatch(eventChan)

	timer := time.NewTimer(time.Millisecond * 100)
	bot.Stop()

	select {
	case v := <-done:
		if v != true {
			t.Error("should be true")
		}
	case <-timer.C:
	}
	wg.Wait()
}

func TestVkBot_Start(t *testing.T) {
	longPollServer := newFakeLongPollServer()
	bot := VkBot{
		handlers: map[string]HandleFunc{},
		longPollServer: longPollServer,
		config: BotConfig{
			Workers: 3,
		},
	}
	u, _ := NewUpdate(typed.Typed{
		"ts": "0",
		"updates": []typed.Typed{
			{
				"type":     event.MessageNewType,
				"object":   typed.Typed{},
				"group_id": 0,
				"event_id": "xoox",
			},
			{
				"type":     event.MessageNewType,
				"object":   typed.Typed{},
				"group_id": 0,
				"event_id": "xoox",
			},
			{
				"type":     event.MessageNewType,
				"object":   typed.Typed{},
				"group_id": 0,
				"event_id": "xoox",
			},
		},
	})

	longPollServer.startUpdatesLoopFunc = func() <-chan Update {
		updatesChan := make(chan Update)
		go func() {
			defer close(updatesChan)
			updatesChan <- u
		}()
		return updatesChan
	}

	wg := &sync.WaitGroup{}
	wg.Add(3)

	bot.EventHandler(event.MessageNewType, func(e event.Event) error {
		wg.Done()
		return nil
	})
	bot.Start()
	wg.Wait()
	bot.Stop()
}
