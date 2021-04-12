package vkbot

import (
	"context"
	"fmt"
	"github.com/AndrewShukhtin/vkbot/event"
	"github.com/karlseguin/typed"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

type (
	// GroupLongPollServer client for api.vk.com groupLongPollServer
	GroupLongPollServer interface {
		// Settings get settings set by SetSettings or default settings of GroupLongPollServer
		// enabled - 1, message_new - 1, others - 0
		Settings() Params

		// SetSettings set GroupLongPollServer settings
		SetSettings(settings Params)

		// SetConfig set GroupLongPollServer wait time for update receiving
		SetConfig(config LongPollConfig)

		// Init initialize GroupLongPollServer and check errors
		Init() error

		// StartUpdatesLoop start receiving update from GroupLongPollServer
		StartUpdatesLoop() <-chan Update

		// StopUpdatesLoop stop receiving update from GroupLongPollServer
		StopUpdatesLoop()
	}

	groupLongPollServer struct {
		Key         string
		Server      string
		Ts          string
		mtx         *sync.Mutex
		VkAPI       VkAPI
		GroupID     int
		eventCtx    context.Context
		eventCancel context.CancelFunc
		settings    Params
		client      *http.Client
		hookDealer  *hookDealer
		config      LongPollConfig
	}
)

// LongPollConfig enable to configure GroupLongPollServe
type LongPollConfig struct {
	// Wait max time (in seconds) to await updates
	Wait int

	// UpdateBufferSize size of Update chan buffer size
	UpdateBufferSize int

	// Limiter rate limiter for incoming updates
	Limiter *rate.Limiter
}

// NewGroupLongPollServer create new GroupLongPollServer with VkAPI wrapper and group id
func NewGroupLongPollServer(vkAPI VkAPI, groupID int) GroupLongPollServer {
	s := &groupLongPollServer{
		VkAPI:      vkAPI,
		GroupID:    groupID,
		mtx:        &sync.Mutex{},
		eventCtx:   context.Background(),
		client:     client,
		hookDealer: defaultHookDealer(),
		config:     defaultLongPollConfig(),
	}
	s.defaultSettings()
	return s
}

func (s *groupLongPollServer) Settings() Params {
	return s.settings
}

func (s *groupLongPollServer) SetSettings(settings Params) {
	for k, v := range settings {
		if _, ok := s.settings[k]; ok {
			s.settings[k] = v
		}
	}
}

func (s *groupLongPollServer) SetConfig(config LongPollConfig) {
	if config.Wait < 1 || config.Wait > 90 {
		config.Wait = 25
	}
	s.config.Wait = config.Wait
	if config.Limiter == nil {
		config.Limiter = defaultRateLimiter()
	}
	s.config.Limiter = config.Limiter
	if config.UpdateBufferSize < 0 || config.UpdateBufferSize > 1000 {
		// default value
		config.UpdateBufferSize = 10
	}
	s.config.UpdateBufferSize = config.UpdateBufferSize
}

func (s *groupLongPollServer) Init() error {
	_, err := s.VkAPI.CallMethod("groups.setLongPollSettings", s.settings)
	if err != nil {
		return err
	}
	return s.init()
}

func (s *groupLongPollServer) StartUpdatesLoop() <-chan Update {
	out := make(chan Update, s.config.UpdateBufferSize)
	s.eventCtx, s.eventCancel = context.WithCancel(s.eventCtx)

	o := newOverHeater(time.Millisecond*50, 3)
	go func(ctx context.Context) {
		defer close(out)
		for {
			if o.isOverHeated() && s.hookDealer.AtOverheat(ctx) {
				// At errors overheat
				return
			}
			if s.config.Limiter.Allow() == false {
				select {
				case <-ctx.Done():
					return
				default:
					s.hookDealer.AtLimit(ctx, s.config.Limiter)
					// At exceed request limit
				}
			}
			in := s.getUpdate()
			select {
			case resp, ok := <-in:
				if !ok {
					return
				}
				if resp.Error != nil {
					// At error response
					s.hookDealer.AtResponseError(o, resp.Error)
					continue
				}
				us, err := NewUpdate(resp.UnpackedResponse)
				if err != nil {
					// At NewUpdate error
					s.hookDealer.AtNewUpdateError(o, err)
					continue
				}
				out <- us
			case <-ctx.Done():
				return
			}
		}
	}(s.eventCtx)
	return out
}

type (
	// Update interface for update
	Update interface {
		// Ts returns timestamp of update
		Ts() string

		// Events returns array of events
		Events() []event.Event
	}

	update struct {
		ts     string
		events []event.Event
	}
)

// NewUpdate parse new update from data
func NewUpdate(data typed.Typed) (Update, error) {
	return parseToUpdateType(data)
}

func (us *update) Ts() string {
	return us.ts
}

func (us *update) Events() []event.Event {
	return us.events
}

func parseToUpdateType(resp typed.Typed) (Update, error) {
	if _, ok := resp["updates"]; !ok {
		return nil, fmt.Errorf("updates field not found")
	}
	us, ok := resp.ObjectsIf("updates")
	if !ok {
		return nil, fmt.Errorf("can not convert update field to []typed.Type")
	}
	if len(us) == 0 {
		return nil, fmt.Errorf("updates field zero length")
	}
	var err error
	res := &update{}
	res.ts = resp.String("ts")
	for _, u := range us {
		e, parseErr := event.NewEvent(u)
		if parseErr != nil {
			err = parseErr
		}
		res.events = append(res.events, e)
	}
	if err != nil {
		return &update{}, err
	}
	return res, nil
}

func (s *groupLongPollServer) StopUpdatesLoop() {
	if s.eventCancel == nil {
		panic("trying to stop not started event loop")
	}
	s.eventCancel()
}

func (s *groupLongPollServer) init() error {
	resp, err := s.VkAPI.CallMethod("groups.getLongPollServer", Params{"group_id": s.GroupID})
	if err != nil {
		return err
	}

	s.mtx.Lock()
	s.Ts = resp.String("ts")
	s.Key = resp.String("key")
	s.Server = resp.String("server")
	s.mtx.Unlock()

	Logger.Info("groupLongPollServer initialized",
		zap.String("ts", s.Ts),
		zap.String("key", s.Key),
		zap.String("server", s.Server))
	return nil
}

type unmarshalledResponseAndErr struct {
	UnpackedResponse typed.Typed
	Error            error
}

func (s *groupLongPollServer) getUpdate() chan unmarshalledResponseAndErr {
	out := make(chan unmarshalledResponseAndErr)
	go func() {
		defer close(out)
		attempts := 5
		for {
			attempts--
			if attempts == 0 {
				out <- unmarshalledResponseAndErr{
					UnpackedResponse: nil,
					Error:            newInternalError(fmt.Errorf("can't making request"), "the maximum number of attempts has been exceeded"),
				}
				return
			}

			params := Params{
				"key":  s.Key,
				"ts":   s.Ts,
				"act":  "a_check",
				"wait": s.config.Wait,
			}
			reqBody := strings.NewReader(params.URLValues().Encode())
			httpReq, err := http.NewRequestWithContext(s.eventCtx, http.MethodPost, s.Server, reqBody)
			if err != nil {
				err := newInternalError(err, "invalid request")
				out <- unmarshalledResponseAndErr{
					UnpackedResponse: nil,
					Error:            err,
				}
				return
			}

			httpResp, err := s.client.Do(httpReq)
			if err != nil {
				err := newInternalError(err, "error occurred while making request")
				out <- unmarshalledResponseAndErr{
					UnpackedResponse: nil,
					Error:            err,
				}
				return
			}
			defer httpResp.Body.Close()

			if httpResp.StatusCode != http.StatusOK {
				continue
			}

			respBody, err := ioutil.ReadAll(httpResp.Body)
			if err != nil {
				out <- unmarshalledResponseAndErr{
					UnpackedResponse: nil,
					Error:            newInternalError(err, "error occurred while reading response body"),
				}
				return
			}

			reply, err := typed.Json(respBody)
			if err != nil {
				out <- unmarshalledResponseAndErr{
					UnpackedResponse: nil,
					Error:            newInternalError(err, "error occurred while unmarshalling"),
				}
				return
			}

			if _, ok := reply["fail"]; ok {
				// TODO: Add switch statement
				if err = s.init(); err != nil {
					out <- unmarshalledResponseAndErr{
						UnpackedResponse: nil,
						Error:            newInternalError(err, "error occurred while re-initialization of long-poll server"),
					}
					return
				}
				continue
			}

			s.mtx.Lock()
			s.Ts = reply.String("ts")
			s.mtx.Unlock()

			out <- unmarshalledResponseAndErr{
				UnpackedResponse: reply,
				Error:            nil,
			}
			return
		}
	}()
	return out
}

func (s *groupLongPollServer) defaultSettings() {
	s.settings = Params{
		"group_id":                         s.GroupID,
		"enabled":                          1,
		"api_version":                      VkAPIVersion,
		"app_payload":                      0,
		"audio_new":                        0,
		"board_post_delete":                0,
		"board_post_edit":                  0,
		"board_post_new":                   0,
		"board_post_restore":               0,
		"group_change_photo":               0,
		"group_change_settings":            0,
		"group_join":                       0,
		"group_leave":                      0,
		"group_officers_edit":              0,
		"market_comment_delete":            0,
		"market_comment_edit":              0,
		"market_comment_new":               0,
		"market_comment_restore":           0,
		"message_allow":                    0,
		"message_deny":                     0,
		"message_new":                      1,
		"message_read":                     0,
		"message_reply":                    0,
		"message_typing_state":             0,
		"message_edit":                     0,
		"photo_comment_delete":             0,
		"photo_comment_edit":               0,
		"photo_comment_new":                0,
		"photo_comment_restore":            0,
		"photo_new":                        0,
		"poll_vote_new":                    0,
		"user_block":                       0,
		"user_unblock":                     0,
		"video_comment_delete":             0,
		"video_comment_edit":               0,
		"video_comment_new":                0,
		"video_comment_restore":            0,
		"video_new":                        0,
		"wall_post_new":                    0,
		"wall_reply_delete":                0,
		"wall_reply_edit":                  0,
		"wall_reply_new":                   0,
		"wall_reply_restore":               0,
		"wall_repost":                      0,
		"lead_forms_new":                   0,
		"like_add":                         0,
		"like_remove":                      0,
		"market_order_new":                 0,
		"market_order_edit":                0,
		"vkpay_transaction":                0,
		"message_event":                    0,
		"donut_subscription_create":        0,
		"donut_subscription_prolonged":     0,
		"donut_subscription_cancelled":     0,
		"donut_subscription_expired":       0,
		"donut_subscription_price_changed": 0,
		"donut_money_withdraw":             0,
		"donut_money_withdraw_error":       0,
	}
}

func defaultRateLimiter() *rate.Limiter {
	return rate.NewLimiter(1, 3)
}

func defaultLongPollConfig() LongPollConfig {
	return LongPollConfig{
		Wait:             25,
		UpdateBufferSize: 10,
		Limiter:          defaultRateLimiter(),
	}
}

func defaultHookDealer() *hookDealer {
	d := &hookDealer{}
	d.AtOverheat = func(ctx context.Context) bool {
		Logger.Error("too mane errors occurred, lets wait several time")
		t := time.NewTimer(time.Second * 3)
		select {
		case <-ctx.Done():
			return true
		case <-t.C:
			return false
		}
	}

	d.AtLimit = func(ctx context.Context, limiter *rate.Limiter) {
		r := limiter.Reserve()
		Logger.Warn(fmt.Sprintf("too many requests, lets wait %v", r.Delay()))
		limiter.Wait(ctx)
	}

	d.AtResponseError = func(o *overHeater, err error) {
		logInternalErrorOr("response with error", err)
		o.addTimeStamp(time.Now())
	}

	d.AtNewUpdateError = func(o *overHeater, err error) {
		Logger.Error("error while unmarshalling update", zap.Error(err))
		o.addTimeStamp(time.Now())
	}
	return d
}

type hookDealer struct {
	AtOverheat       func(context.Context) bool
	AtLimit          func(context.Context, *rate.Limiter)
	AtResponseError  func(*overHeater, error)
	AtNewUpdateError func(*overHeater, error)
}
