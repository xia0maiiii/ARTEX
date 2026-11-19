package db

import (
	"testing"
	"time"
)

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestFrontierSignalRegistry(t *testing.T) {
	const a, b = int64(-101), int64(-102) // ids no real exploration uses
	chA := FrontierChanged(a)
	if FrontierChanged(a) != chA {
		t.Fatal("waiters on the same exploration must share one channel until it fires")
	}
	chB := FrontierChanged(b)
	notifyFrontier(b)
	if !isClosed(chB) || isClosed(chA) {
		t.Fatal("a notification must wake only its own exploration")
	}
	notifyFrontier(a)
	if !isClosed(chA) {
		t.Fatal("notify did not close the waiting channel")
	}
	if next := FrontierChanged(a); isClosed(next) {
		t.Fatal("after firing, the next wait must get a fresh open channel")
	}
	notifyFrontier(a)
	notifyFrontier(a) // no waiter: must be a no-op, not a double close
}

// Writes that make an intent claimable must wake waiters; writes that don't
// must not, so idle workers are not woken for nothing.
func TestFrontierSignalFiresOnClaimableIntent(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()
	expID, err := d.CreateExploration("frontier signal", "goal")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Exec(`DELETE FROM explorations WHERE id=$1`, expID) //nolint:errcheck
	es := d.Exploration(expID)

	wake := FrontierChanged(expID)
	if _, err := es.AddGoal(map[string]any{"text": "x"}, "human"); err != nil { // state open, but not an intent
		t.Fatal(err)
	}
	if isClosed(wake) {
		t.Fatal("an open non-intent node must not wake workers")
	}

	intent, err := es.AddIntent(map[string]any{"summary": "probe"}, 1, nil, "planner")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-wake:
	case <-time.After(time.Second):
		t.Fatal("AddIntent did not wake waiters")
	}

	if ok, err := es.ClaimIntent(intent, "w1"); err != nil || !ok {
		t.Fatalf("claim: ok=%v err=%v", ok, err)
	}
	wake = FrontierChanged(expID)
	if ok, err := es.CompareAndSetIntentState(intent, "running", "done"); err != nil || !ok {
		t.Fatalf("finish: ok=%v err=%v", ok, err)
	}
	if isClosed(wake) {
		t.Fatal("finishing an intent must not wake workers")
	}

	if ok, err := es.ReopenIntent(intent); err != nil || ok {
		t.Fatalf("done intents are not reopenable: ok=%v err=%v", ok, err)
	}
	if err := es.SetIntentState(intent, "blocked"); err != nil {
		t.Fatal(err)
	}
	if isClosed(wake) {
		t.Fatal("blocking an intent must not wake workers")
	}
	if ok, err := es.ReopenIntent(intent); err != nil || !ok {
		t.Fatalf("reopen blocked: ok=%v err=%v", ok, err)
	}
	if !isClosed(wake) {
		t.Fatal("ReopenIntent did not wake waiters")
	}
}
