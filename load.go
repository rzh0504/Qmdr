package main

import (
	"context"
	"sync/atomic"
	"time"
)

// asyncSlot guards one family of async fetches: every fetch claims a
// sequence on start, and only the newest claim's result reaches the UI.
type asyncSlot struct {
	seq atomic.Int64
}

// reset invalidates the fetches in flight: their results will not apply.
func (s *asyncSlot) reset() { s.seq.Add(1) }

// async runs fetch on its own goroutine with a timeout, then apply on the
// window's thread — but only while no newer fetch of the slot has started.
// fetch writes its result into variables the two closures share, so no
// result type needs a name; apply turns that data into state changes.
func (a *app) async(slot *asyncSlot, timeout time.Duration, fetch func(ctx context.Context) error, apply func(err error)) {
	sequence := slot.seq.Add(1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := fetch(ctx)
		a.update(func() {
			if slot.seq.Load() != sequence {
				return
			}
			apply(err)
		})
	}()
}
