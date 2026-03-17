package throttle

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type entry struct {
	lastUnixNano int64
}

type shard struct {
	mu sync.Mutex
	m  map[int]entry
}

type LocalThrottle struct {
	shards  [256]shard
	window  int64 // ns
	started atomic.Bool
	stopCh  chan struct{}
}

func NewLocalThrottle(window time.Duration) *LocalThrottle {
	t := &LocalThrottle{
		window: int64(window),
		stopCh: make(chan struct{}),
	}
	for i := range t.shards {
		t.shards[i].m = make(map[int]entry)
	}
	t.StartJanitor(context.Background(), 30*time.Second, 5*time.Minute)
	return t
}

// Allow returns true if key is allowed to run now; false if throttled.
func (t *LocalThrottle) Allow(key int) bool {
	s := &t.shards[uint8(key)]
	now := time.Now().UnixNano()

	s.mu.Lock()
	last := s.m[key].lastUnixNano
	if last != 0 && now-last < t.window {
		s.mu.Unlock()
		return false
	}
	s.m[key] = entry{lastUnixNano: now}
	s.mu.Unlock()
	return true
}

// StartJanitor starts a background cleanup loop.
// interval: how often to scan
// gcAfter: delete keys not touched for longer than this duration (建议远大于 window，比如 1-10 分钟)
func (t *LocalThrottle) StartJanitor(ctx context.Context, interval, gcAfter time.Duration) {
	if !t.started.CompareAndSwap(false, true) {
		return
	}
	gcAfterNs := int64(gcAfter)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.stopCh:
				return
			case <-ticker.C:
				now := time.Now().UnixNano()
				for i := range t.shards {
					s := &t.shards[i]
					s.mu.Lock()
					for k, e := range s.m {
						if now-e.lastUnixNano > gcAfterNs {
							delete(s.m, k)
						}
					}
					s.mu.Unlock()
				}
			}
		}
	}()
}

func (t *LocalThrottle) Stop() {
	select {
	case <-t.stopCh:
		// already closed
	default:
		close(t.stopCh)
	}
}
