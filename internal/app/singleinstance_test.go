package app

import (
	"testing"
	"time"
)

// TestSingleInstance guards the click-again behavior: the first holder
// wins, a second holder loses while signalling the first to raise, and
// the lock is free again after release.
func TestSingleInstance(t *testing.T) {
	raised := make(chan struct{}, 1)
	bound, release, first := acquireSingleInstance("127.0.0.1:0", func() { raised <- struct{}{} })
	if !first {
		t.Fatal("first acquire should win")
	}
	if bound == "" {
		release()
		t.Fatal("no bound address returned")
	}

	_, _, second := acquireSingleInstance(bound, func() {})
	if second {
		release()
		t.Fatal("second acquire should lose")
	}
	select {
	case <-raised:
	case <-time.After(2 * time.Second):
		release()
		t.Fatal("running instance was not signalled")
	}

	release()
	deadline := time.Now().Add(2 * time.Second)
	for {
		_, release3, third := acquireSingleInstance(bound, func() {})
		if third {
			release3()
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("lock not released after close")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
