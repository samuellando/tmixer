package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type wideEvent struct {
	mu       *sync.Mutex
	level    string
	time     time.Time
	duration time.Duration
	error    error
	events   []*event
}

func newWideEvent() wideEvent {
	return wideEvent{mu: &sync.Mutex{}, level: "INFO", time: time.Now()}
}

func (w *wideEvent) fatal(err error) {
	w.level = "ERROR"
	w.error = err
}

func (w *wideEvent) add(e *event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.events = append(w.events, e)
}

func (w wideEvent) MarshalJSON() ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := bytes.Buffer{}
	out.WriteByte('{')
	// Level
	fmt.Fprintf(&out, `"%s":%s`, "level", w.level)
	out.WriteByte(',')
	// Time
	b, err := json.Marshal(w.time)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&out, `"%s":%s`, "time", string(b))
	out.WriteByte(',')
	// error
	if w.error != nil {
		b, err := json.Marshal(w.error)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, `"%s":%s`, "error", string(b))
		out.WriteByte(',')
	}
	// Events
	for _, event := range w.events {
		b, err = json.Marshal(event)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, `"%s":%s`, event.name, string(b))
		out.WriteByte(',')
	}
	// Duration
	b, err = json.Marshal(w.duration)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&out, `"%s":%s`, "duration", string(b))
	out.WriteByte('}')
	return out.Bytes(), nil
}
