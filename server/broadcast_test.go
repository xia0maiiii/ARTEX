package server

import (
	"testing"

	"github.com/Autumn-27/artex/db"
)

func TestBroadcasterDeliversInOrder(t *testing.T) {
	b := NewBroadcaster()
	ch, unsub := b.Subscribe("1")
	defer unsub()
	for i := int64(1); i <= 3; i++ {
		b.Publish("1", db.Activity{ID: i})
	}
	b.Publish("2", db.Activity{ID: 99}) // other task: not delivered
	for want := int64(1); want <= 3; want++ {
		if got := (<-ch).ID; got != want {
			t.Fatalf("got id %d, want %d", got, want)
		}
	}
	select {
	case a := <-ch:
		t.Fatalf("unexpected extra event %d", a.ID)
	default:
	}
}

// A lagging subscriber must be disconnected rather than silently skipping an
// event: the SSE handler returns on a closed channel and the client resumes from
// the DB with Last-Event-ID, so no id is ever permanently missing.
func TestBroadcasterClosesLaggingSubscriber(t *testing.T) {
	b := NewBroadcaster()
	slow, unsubSlow := b.Subscribe("1")
	fast, unsubFast := b.Subscribe("1")
	defer unsubFast()

	for i := int64(1); i <= subscriberBuffer; i++ {
		b.Publish("1", db.Activity{ID: i})
		<-fast // keep the fast subscriber drained
	}
	b.Publish("1", db.Activity{ID: subscriberBuffer + 1}) // overflows slow only
	if got := (<-fast).ID; got != subscriberBuffer+1 {
		t.Fatalf("fast subscriber got %d, want %d", got, subscriberBuffer+1)
	}

	var last int64
	for a := range slow { // terminates only if the channel was closed
		last = a.ID
	}
	if last != subscriberBuffer {
		t.Fatalf("slow subscriber drained up to %d, want %d buffered events", last, subscriberBuffer)
	}

	unsubSlow() // must not double-close
	unsubSlow()
	b.Publish("1", db.Activity{ID: subscriberBuffer + 2}) // must not panic on the removed subscriber
	if got := (<-fast).ID; got != subscriberBuffer+2 {
		t.Fatalf("fast subscriber got %d after slow removal", got)
	}
}
