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

func BuildFirstKeyboard() *keyboard.Keyboard {
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

func BuildSecondKeyboard() *keyboard.Keyboard {
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

type BotApp struct {
	vkBot *vkbot.VkBot
	vkApi vkbot.VkApi
	menus map[string]*keyboard.Keyboard
}

func NewBotApp(token string, groupId int) *BotApp {
	vkApi := vkbot.NewVkApi(token)
	longPollServer := vkbot.NewGroupLongPollServer(vkApi, groupId)
	longPollServer.SetSettings(vkbot.Params{"message_event": 1})
	return &BotApp{vkBot: vkbot.NewVkBot(vkApi, longPollServer), vkApi: vkApi}
}

func (app *BotApp) MessageEventHandler(e event.Event) error {
	me := e.Object()
	t := typed.New(me.Object("payload"))

	if t.String("type") == "go_to_second" {
		k, _ := app.menus["second"].Json()
		_, err := app.vkApi.CallMethod("messages.edit",
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
		k, _ := app.menus["first"].Json()
		_, err := app.vkApi.CallMethod("messages.edit",
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

func (app *BotApp) MessageNewHandler(e event.Event) error {
	mn := e.Object()
	m := mn.Object("message")
	if m.String("text") == "first" {
		k, _ := app.menus["first"].Json()
		_, err := app.vkApi.CallMethod("messages.send",
			vkbot.Params{
				"peer_id":                 m.Int("peer_id"),
				"random_id":               time.Now().UnixNano(),
				"message":                 "first keyboard",
				"conversation_message_id": m.Int("conversation_message_id"),
				"keyboard":                k,
			})
		return err
	}
	return nil
}

func (app *BotApp) Init() error {
	app.menus = make(map[string]*keyboard.Keyboard, 2)
	app.menus["first"] = BuildFirstKeyboard()
	app.menus["second"] = BuildSecondKeyboard()

	app.vkBot.EventHandler(event.MessageNewType, app.MessageNewHandler)
	app.vkBot.EventHandler(event.MessageEventType, app.MessageEventHandler)
	return app.vkBot.Init()
}

func (app *BotApp) Start() {
	app.vkBot.Start()
}

func (app *BotApp) Stop() {
	app.vkBot.Stop()
}

func main() {
	GroupId, _ := strconv.Atoi(os.Getenv("VK_GROUP_ID"))
	Token := os.Getenv("VK_GROUP_TOKEN")
	app := NewBotApp(Token, GroupId)

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
