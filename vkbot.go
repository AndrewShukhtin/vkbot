package vkbot

import (
	"fmt"
	"github.com/AndrewShukhtin/vkbot/event"
	"github.com/fatih/color"
	"go.uber.org/zap"
)

const (
	// Version of framework
	Version = "0.0.1"
	banner  = `
 ___      ___ ___  __    ________  ________  _________   
|\  \    /  /|\  \|\  \ |\   __  \|\   __  \|\___   ___\ 
\ \  \  /  / \ \  \/  /|\ \  \|\ /\ \  \|\  \|___ \  \_| 
 \ \  \/  / / \ \   ___  \ \   __  \ \  \\\  \   \ \  \  
  \ \    / /   \ \  \\ \  \ \  \|\  \ \  \\\  \   \ \  \ 
   \ \__/ /     \ \__\\ \__\ \_______\ \_______\   \ \__\
    \|__|/       \|__| \|__|\|_______|\|_______|    \|__| v %s

                   vk.com group bot framework                  /
---------------------------------------------------------------
                                                               \
`
)

// HandleFunc alias for event handler function
type HandleFunc func(event.Event) error

var notFoundHandler HandleFunc = func(e event.Event) error {
	return fmt.Errorf("not implemented event handler for '%s' event", e.Type())
}

// BotConfig allows to configure bot
type BotConfig struct {
	// Workers number of workers
	Workers int

	// WorkerBuffer buffer of each worker chan
	WorkerBuffer int

	// Events buffer off events chan
	Events int
}

// VkBot structure for handle events from GroupLongPollServer
type VkBot struct {
	vkAPI          VkAPI
	longPollServer GroupLongPollServer
	handlers       map[string]HandleFunc

	config     BotConfig
	dispatcher *dispatcher

	enableBanner bool
}

// NewVkBot creates new VkBot
func NewVkBot(vkAPI VkAPI, longPollServer GroupLongPollServer) *VkBot {
	b := &VkBot{
		vkAPI:          vkAPI,
		longPollServer: longPollServer,
		handlers:       make(map[string]HandleFunc),
		config:         defaultConfig(),
		enableBanner:   true,
	}
	return b
}

// EventHandler allows to add handler for event type
func (bot *VkBot) EventHandler(eventType string, handler HandleFunc) {
	bot.handlers[eventType] = handler
}

// SetConfig sets configuration of bot
func (bot *VkBot) SetConfig(cfg BotConfig) {
	bot.config = cfg
}

// Init initializes longPollServer and check correctness of handlers
func (bot *VkBot) Init() error {
	if bot.enableBanner {
		c := color.New(color.FgBlue, color.Bold)
		c.Printf(banner, Version)
	}
	settings := bot.longPollServer.Settings()
	for k, h := range bot.handlers {
		if _, ok := settings[k]; !ok {
			return fmt.Errorf("added handler for unsupported event type %s", k)
		}
		if h == nil {
			return fmt.Errorf("nil handler for %s event", k)
		}
	}
	err := bot.longPollServer.Init()
	if err != nil {
		return err
	}
	Logger.Info("VkBot initialized")
	return nil
}

// Start serves the incoming events
func (bot *VkBot) Start() {
	bot.dispatcher = newDispatcher(bot.config.Workers, bot.config.WorkerBuffer)
	bot.dispatcher.setWorkerFunc(bot.handleEvent)
	bot.dispatcher.startWorkers()
	updatesChan := bot.longPollServer.StartUpdatesLoop()
	eventsChan := make(chan event.Event, bot.config.Events)
	go func() {
		defer close(eventsChan)
		for u := range updatesChan {
			for _, e := range u.Events() {
				eventsChan <- e
			}
		}
	}()
	bot.dispatcher.dispatch(eventsChan)
}

// Stop stops serving incoming events
func (bot *VkBot) Stop() {
	bot.longPollServer.StopUpdatesLoop()
	bot.dispatcher.stopWorkers(func() { /*dumb hook*/ })
}

func (bot *VkBot) handleEvent(e event.Event) {
	var handler = notFoundHandler
	if h, ok := bot.handlers[e.Type()]; ok {
		handler = h
	}
	err := handler(e)
	Logger.Info(fmt.Sprintf("handled '%s' event", e.Type()))
	if err != nil {
		Logger.Error("something went wrong", zap.Error(err))
	}
}

type workerFunc func(event.Event)

type worker struct {
	workersPool  chan chan event.Event
	dataChannel  chan event.Event
	workerBuffer int
	done         <-chan struct{}
	workerFunc   workerFunc
}

func newWorker(workersPool chan chan event.Event, workerBuffer int, done <-chan struct{}) *worker {
	return &worker{
		workersPool:  workersPool,
		done:         done,
		workerBuffer: workerBuffer,
	}
}

func (w *worker) setWorkerFunc(wf workerFunc) {
	w.workerFunc = wf
}

func (w *worker) run() {
	w.dataChannel = make(chan event.Event, w.workerBuffer)
	go func() {
		defer close(w.dataChannel)
		for {
			w.workersPool <- w.dataChannel
			select {
			case e, ok := <-w.dataChannel:
				if !ok {
					return
				}
				w.workerFunc(e)
			case <-w.done:
				return
			}
		}
	}()
}

type dispatcher struct {
	workersPool    chan chan event.Event
	workerPoolSize int
	workerBuffer   int
	done           chan struct{}
	workerFunc     workerFunc
}

func newDispatcher(workersPoolSize int, workerBuffer int) *dispatcher {
	pool := make(chan chan event.Event, workersPoolSize)
	return &dispatcher{
		workersPool:    pool,
		workerPoolSize: workersPoolSize,
		workerBuffer:   workerBuffer,
		done:           make(chan struct{}),
	}
}

func (d *dispatcher) setWorkerFunc(wf workerFunc) {
	d.workerFunc = wf
}

func (d *dispatcher) startWorkers() {
	d.done = make(chan struct{})
	for i := 0; i < d.workerPoolSize; i++ {
		w := newWorker(d.workersPool, d.workerBuffer, d.done)
		w.setWorkerFunc(d.workerFunc)
		w.run()
	}
}

func (d *dispatcher) stopWorkers(hook hookFunc) {
	go func() {
		close(d.done)
		hook()
	}()
}

func (d *dispatcher) dispatch(eventChan <-chan event.Event) {
	for {
		select {
		case data, ok := <-eventChan:
			if !ok {
				return
			}
			go func(data event.Event) {
				dataChannel := <-d.workersPool
				dataChannel <- data
			}(data)
		case <-d.done:
			return
		}
	}
}

func defaultConfig() BotConfig {
	return BotConfig{
		Workers:      16,
		WorkerBuffer: 4,
		Events:       16,
	}
}

type hookFunc func()
