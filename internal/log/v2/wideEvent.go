package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"
)

type wideEvent struct {
	mu       *sync.Mutex
	level    string
	time     time.Time
	duration time.Duration
	error    error
	events   map[string]*event
}

func newWideEvent() wideEvent {
	return wideEvent{
		mu:     &sync.Mutex{},
		level:  "INFO",
		time:   time.Now(),
		events: make(map[string]*event),
	}
}

func (w *wideEvent) fatal(err error) {
	w.level = "ERROR"
	w.error = err
}

func (w *wideEvent) add(e *event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.events[e.name] = e
}

func (w *wideEvent) done() {
	if w.duration == 0 {
		w.duration = time.Since(w.time)
	}
	for _, event := range w.events {
		event.Done()
	}
}

func (w wideEvent) MarshalJSON() ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := bytes.Buffer{}
	out.WriteByte('{')
	// Level
	b, err := json.Marshal(w.level)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&out, `"%s":%s`, "level", string(b))
	out.WriteByte(',')
	// Time
	b, err = json.Marshal(w.time)
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
	for _, event := range w.sortEventsByTimestamp() {
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

func (w wideEvent) sortEventsByTimestamp() []*event {
	l := make([]*event, 0, len(w.events))
	for _, event := range w.events {
		l = append(l, event)
	}
	slices.SortFunc(l, func(a, b *event) int {
		return int(a.time.Sub(b.time))
	})
	return l
}
