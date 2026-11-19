package server

import (
	"sync"

	"github.com/Autumn-27/artex/db"
)

// Broadcaster is a per-task in-process pub/sub for live activity events. The
// engine publishes each appended activity at its single emit point; SSE handlers
// subscribe per task. Storage (activity table) and the live stream come from the
// same Publish call, so they never diverge.
type Broadcaster struct {
	mu   sync.Mutex
	subs map[string]map[*subscriber]struct{} // task id -> set of subscribers
}

// subscriber owns one buffered channel. It is closed exactly once, either by the
// caller's unsubscribe or by Publish when the subscriber falls behind.
type subscriber struct {
	ch   chan db.Activity
	once sync.Once
}

func (s *subscriber) close() { s.once.Do(func() { close(s.ch) }) }

const subscriberBuffer = 256

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{subs: map[string]map[*subscriber]struct{}{}}
}

// Subscribe returns a buffered channel of activities for a task plus an
// unsubscribe func the caller must invoke (defer) to release it. The channel is
// also closed if the subscriber lags (see Publish); callers must treat a closed
// channel as "resume from the DB using the last delivered id".
func (b *Broadcaster) Subscribe(task string) (<-chan db.Activity, func()) {
	sub := &subscriber{ch: make(chan db.Activity, subscriberBuffer)}
	b.mu.Lock()
	if b.subs[task] == nil {
		b.subs[task] = map[*subscriber]struct{}{}
	}
	b.subs[task][sub] = struct{}{}
	b.mu.Unlock()

	return sub.ch, func() {
		b.mu.Lock()
		b.removeLocked(task, sub)
		b.mu.Unlock()
		sub.close()
	}
}

func (b *Broadcaster) removeLocked(task string, sub *subscriber) {
	if m := b.subs[task]; m != nil {
		delete(m, sub)
		if len(m) == 0 {
			delete(b.subs, task)
		}
	}
}

// Publish fans an activity out to all subscribers of a task without blocking the
// engine. A subscriber whose buffer is full is dropped and its channel closed:
// silently skipping the event would leave a permanent hole, because later events
// carry higher ids and the client's resume cursor would jump past the gap. Closing
// ends the SSE response, the browser's EventSource reconnects with Last-Event-ID,
// and the stream handler replays everything after it from the activity table.
func (b *Broadcaster) Publish(task string, a db.Activity) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for sub := range b.subs[task] {
		select {
		case sub.ch <- a:
		default:
			b.removeLocked(task, sub)
			sub.close()
		}
	}
}
