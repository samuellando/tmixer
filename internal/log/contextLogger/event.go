package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Event interface {
	// Add a specific value to the event
	Log(string, any)
	// Mark the event as done
	Done()
	// Add an error the the event
	Error(error)
}

type event struct {
	mu       sync.Mutex
	name     string
	time     time.Time
	duration time.Duration
	errors   []error
	data     []value
}

type value struct {
	key   string
	value any
}

func newEvent(name string) *event {
	return &event{mu: sync.Mutex{}, name: name, time: time.Now()}
}

func (e *event) Error(err error) {
	e.errors = append(e.errors, err)
}

func (e *event) Done() {
	if e.duration == 0 {
		e.duration = time.Since(e.time)
	}
}

func (e *event) Log(k string, v any) {
	e.data = append(e.data, value{key: k, value: v})
}

func (e *event) MarshalJSON() ([]byte, error) {
	e.Done()
	e.mu.Lock()
	defer e.mu.Unlock()
	out := bytes.Buffer{}
	out.WriteByte('{')
	// Time
	b, err := json.Marshal(e.time)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&out, `"%s":%s`, "time", string(b))
	out.WriteByte(',')
	// data
	for _, v := range e.data {
		b, err = json.Marshal(v.value)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, `"%s":%s`, v.key, string(b))
		out.WriteByte(',')
	}
	// errors
	if len(e.errors) != 0 {
		b, err := json.Marshal(e.errors)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, `"%s":%s`, "errors", string(b))
		out.WriteByte(',')
	}
	// Duration
	b, err = json.Marshal(e.duration)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&out, `"%s":%s`, "duration", string(b))
	out.WriteByte('}')
	return out.Bytes(), nil
}
