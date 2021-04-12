package vkbot

import (
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestParams(t *testing.T) {
	expected := url.Values{}
	expected.Set("test1", "1")

	params := Params{
		"test1": 1,
		"no_conversion": Params{"test": 1},
	}

	res := params.URLValues()
	if res.Encode() != expected.Encode() {
		t.Errorf("results should be equal:\nresult:\n%v\nexpected:\n%v\n", res, expected)
	}
}

func TestOverHeaterWithoutHeat(t *testing.T) {
	o := newOverHeater(time.Second, 3)
	for i := 0; i < 2; i++ {
		o.addTimeStamp(time.Now())
		if o.isOverHeated() {
			t.Error("should not be overheated")
		}
	}
	o.addTimeStamp(time.Now())
	if o.isOverHeated() {
		t.Error("should be not overheated")
	}
}

func TestOverHeaterSequential(t *testing.T) {
	o := newOverHeater(time.Second, 2)
	for i := 0; i < 2; i++ {
		o.addTimeStamp(time.Now())
		if o.isOverHeated() {
			t.Error("should not be overheated")
		}
	}
	// from this moment its overheated
	o.addTimeStamp(time.Now())
	if !o.isOverHeated() {
		t.Error("should be overheated")
	}
}

func TestOverHeaterConcurrent(t *testing.T) {
	o := newOverHeater(time.Second, 2)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			o.addTimeStamp(time.Now())
		}()
	}
	wg.Wait()
	// from this moment its overheated
	o.addTimeStamp(time.Now())
	if !o.isOverHeated() {
		t.Error("should be overheated")
	}
}