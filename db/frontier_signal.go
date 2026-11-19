package db

import "sync"

// frontierSignals lets idle workers sleep until their exploration may have
// gained a claimable intent, instead of re-querying the frontier on a short
// timer. Each waiting exploration has one channel that is closed (and dropped)
// on the next change, so every waiter wakes once per change.
//
// The signal is process-local and best-effort: every write in this package that
// creates an open intent or moves one back to 'open' fires it, and workers keep
// a slow fallback poll for anything that bypasses it (raw SQL, another process
// sharing the database). A missed signal only costs latency, never work.
var frontierSignals = struct {
	mu    sync.Mutex
	chans map[int64]chan struct{}
}{chans: map[int64]chan struct{}{}}

// FrontierChanged returns a channel that is closed the next time exploration
// expID may have gained a claimable intent. Callers must obtain it BEFORE
// checking the frontier, so a change that lands between the check and the wait
// is not missed.
func FrontierChanged(expID int64) <-chan struct{} {
	frontierSignals.mu.Lock()
	defer frontierSignals.mu.Unlock()
	ch := frontierSignals.chans[expID]
	if ch == nil {
		ch = make(chan struct{})
		frontierSignals.chans[expID] = ch
	}
	return ch
}

// notifyFrontier wakes everything waiting on expID's frontier. Call it only
// after the write is committed, or a woken worker may not see the new row yet.
func notifyFrontier(expID int64) {
	frontierSignals.mu.Lock()
	defer frontierSignals.mu.Unlock()
	if ch := frontierSignals.chans[expID]; ch != nil {
		close(ch)
		delete(frontierSignals.chans, expID)
	}
}
