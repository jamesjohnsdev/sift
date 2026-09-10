package provider

import (
	"context"
	"time"
)

// PollWatch is a Watch() implementation for providers with no push
// mechanism a desktop app can host (both Gmail and Outlook require a
// public webhook/Pub-Sub endpoint for real push). It polls fetch on
// interval, emitting a MessageAdded update for every id not seen on a
// previous poll, until ctx is done.
func PollWatch(ctx context.Context, interval time.Duration, fetch func(ctx context.Context) ([]MessageID, error)) <-chan Update {
	ch := make(chan Update)

	go func() {
		defer close(ch)
		seen := make(map[MessageID]bool)

		poll := func() bool {
			ids, err := fetch(ctx)
			if err != nil {
				return true // transient fetch error; try again next tick
			}
			for _, id := range ids {
				if seen[id] {
					continue
				}
				seen[id] = true
				select {
				case ch <- Update{Kind: MessageAdded, MessageID: id}:
				case <-ctx.Done():
					return false
				}
			}
			return true
		}

		if !poll() {
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !poll() {
					return
				}
			}
		}
	}()

	return ch
}
