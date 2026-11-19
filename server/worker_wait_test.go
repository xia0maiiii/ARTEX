package server

import (
	"context"
	"testing"
	"time"
)

func TestWaitFrontierWakesOnSignalNotTimer(t *testing.T) {
	wake := make(chan struct{})
	go func() {
		time.Sleep(20 * time.Millisecond)
		close(wake)
	}()
	start := time.Now()
	if done := waitFrontier(context.Background(), wake, time.Minute); done {
		t.Fatal("a frontier signal must not report ctx done")
	}
	if waited := time.Since(start); waited > 2*time.Second {
		t.Fatalf("worker slept %v instead of waking on the signal", waited)
	}
}

func TestWaitFrontierFallbackAndCancel(t *testing.T) {
	never := make(chan struct{})
	if done := waitFrontier(context.Background(), never, 10*time.Millisecond); done {
		t.Fatal("the fallback timer must not report ctx done")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if done := waitFrontier(ctx, never, time.Minute); !done {
		t.Fatal("a cancelled ctx must stop the worker")
	}
}
