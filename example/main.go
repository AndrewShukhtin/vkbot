package main

import (
	"fmt"
	"github.com/AndrewShukhtin/vkbot"
	"github.com/AndrewShukhtin/vkbot/event"
	"github.com/AndrewShukhtin/vkbot/keyboard"
	"github.com/karlseguin/typed"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func buildFirstKeyboard() *keyboard.Keyboard {
	a1 := keyboard.NewTextAction("button 1")
	a1.SetPayload(vkbot.Params{"cmd": "button 1"})

	a2 := keyboard.NewTextAction("button 2")
	a2.SetPayload(vkbot.Params{"cmd": "button 2"})

	a3 := keyboard.NewCallbackAction("second keyboard")
	a3.SetPayload(vkbot.Params{"type": "go_to_second"})
	k := keyboard.NewKeyboard(false, true)

	k.AddButton(keyboard.NewButton(a1, "secondary"))
	k.AddButton(keyboard.NewButton(a2, "secondary"))
	k.AddButton(keyboard.NewButton(a3, "positive"))

	return k
}

func buildSecondKeyboard() *keyboard.Keyboard {
	a1 := keyboard.NewTextAction("button 3")
	a1.SetPayload(vkbot.Params{"cmd": "button 3"})

	a2 := keyboard.NewTextAction("button 4")
	a2.SetPayload(vkbot.Params{"cmd": "button 4"})

	a3 := keyboard.NewCallbackAction("first keyboard")
	a3.SetPayload(vkbot.Params{"type": "go_to_first"})

	k := keyboard.NewKeyboard(false, true)
	k.AddButton(keyboard.NewButton(a1, "secondary"))
	k.AddButton(keyboard.NewButton(a2, "secondary"))
	k.AddButton(keyboard.NewButton(a3, "positive"))

	return k
}

// BotApp example bot application
type BotApp struct {
	vkBot *vkbot.VkBot
	vkAPI vkbot.VkAPI
	menus map[string]*keyboard.Keyboard
}

// NewBotApp new bot app with token and group_id
func NewBotApp(token string, groupID int) *BotApp {
	vkAPI := vkbot.NewVkAPI(token)
	longPollServer := vkbot.NewGroupLongPollServer(vkAPI, groupID)
	longPollServer.SetSettings(vkbot.Params{"message_event": 1})
	return &BotApp{vkBot: vkbot.NewVkBot(vkAPI, longPollServer), vkAPI: vkAPI}
}

// MessageEventHandler handler for message_event
func (app *BotApp) MessageEventHandler(e event.Event) error {
	me := e.Object()
	t := typed.New(me.Object("payload"))

	if t.String("type") == "go_to_second" {
		k, _ := app.menus["second"].JSON()
		_, err := app.vkAPI.CallMethod("messages.edit",
			vkbot.Params{
				"peer_id":                 me.Int("peer_id"),
				"message":                 "second keyboard",
				"conversation_message_id": me.Int("conversation_message_id"),
				"keyboard":                k,
			})
		if err != nil {
			return err
		}
	}
	if t.String("type") == "go_to_first" {
		k, _ := app.menus["first"].JSON()
		_, err := app.vkAPI.CallMethod("messages.edit",
			vkbot.Params{
				"peer_id":                 me.Int("peer_id"),
				"message":                 "first keyboard",
				"conversation_message_id": me.Int("conversation_message_id"),
				"keyboard":                k,
			})
		if err != nil {
			return err
		}
	}
	return nil
}

// MessageNewHandler handler for message_new
func (app *BotApp) MessageNewHandler(e event.Event) error {
	mn := e.Object()
	m := mn.Object("message")
	if m.String("text") == "go" {
		k, _ := app.menus["first"].JSON()
		_, err := app.vkAPI.CallMethod("messages.send",
			vkbot.Params{
				"peer_id":                 m.Int("peer_id"),
				"random_id":               getRandomID(),
				"message":                 "first keyboard",
				"conversation_message_id": m.Int("conversation_message_id"),
				"keyboard":                k,
			})
		return err
	}
	return nil
}

// Init initializes bot app
func (app *BotApp) Init() error {
	app.menus = make(map[string]*keyboard.Keyboard, 2)
	app.menus["first"] = buildFirstKeyboard()
	app.menus["second"] = buildSecondKeyboard()

	app.vkBot.EventHandler(event.MessageNewType, app.MessageNewHandler)
	app.vkBot.EventHandler(event.MessageEventType, app.MessageEventHandler)
	return app.vkBot.Init()
}

// Start app
func (app *BotApp) Start() {
	app.vkBot.Start()
}

// Stop app
func (app *BotApp) Stop() {
	app.vkBot.Stop()
}

func main() {
	GroupID, _ := strconv.Atoi(os.Getenv("VK_GROUP_ID"))
	Token := os.Getenv("VK_GROUP_TOKEN")
	app := NewBotApp(Token, GroupID)

	if err := app.Init(); err != nil {
		vkbot.Logger.Fatal("initialization error", zap.Error(err))
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool)

	go func() {
		sig := <-sigChan
		app.Stop()
		vkbot.Logger.Sync()
		fmt.Println()
		fmt.Println("Caught", sig, "signal")
		fmt.Print("Gracefully stop")
		for range []int{1, 2, 3} {
			time.Sleep(800 * time.Millisecond)
			fmt.Print(".")
		}
		time.Sleep(400 * time.Millisecond)
		fmt.Println()
		close(done)
	}()

	app.Start()

	<-done
	close(sigChan)
}

func getRandomID() int64 {
	return time.Now().UnixNano()
}