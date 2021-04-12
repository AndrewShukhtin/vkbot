package vkbot

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)

// Params allows you to pass keys with values of various types
// It support only int and string
type Params map[string]interface{}

// URLValues convert Params to url.Values
// Do not skipped only string or numeric fields
func (p Params) URLValues() url.Values {
	values := url.Values{}
	for name, i := range p {
		switch i.(type) {
		case string, int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
		default:
			continue
		}
		values.Set(name, fmt.Sprintf("%v", i))
	}
	return values
}

type overHeater struct {
	threshold      time.Duration
	counter        int
	capacity       int
	firstTimeStamp time.Time
	lastTimeStamp  time.Time
	mtx            *sync.Mutex
}

func newOverHeater(threshold time.Duration, capacity int) *overHeater {
	return &overHeater{
		threshold: threshold,
		capacity:  capacity,
		counter:   capacity,
		mtx:       &sync.Mutex{},
	}
}

func (o *overHeater) addTimeStamp(ts time.Time) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	if o.capacity == o.counter {
		o.firstTimeStamp = ts
		o.counter--
		return
	}
	o.lastTimeStamp = ts
	o.counter--
}

func (o *overHeater) isOverHeated() bool {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	if o.counter >= 0 {
		return false
	}

	flag := o.lastTimeStamp.Sub(o.firstTimeStamp).Nanoseconds() < o.threshold.Nanoseconds()
	if flag {
		o.counter = o.capacity
		return true
	}
	return false
}
