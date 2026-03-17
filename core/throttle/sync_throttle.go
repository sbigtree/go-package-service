package throttle

import (
	"context"
	"fmt"
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"
)

type _syncEntry struct {
	mu           sync.Mutex
	cond         *sync.Cond
	next         uint64 // 下一个
	serving      uint64 // 正在执行的
	lastUnixNano int64
}

type _shard[K comparable] struct {
	mu sync.Mutex
	m  map[K]*_syncEntry
}

type LocalSyncThrottle[K comparable] struct {
	shards  [256]_shard[K]
	shardFn func(K) uint8
	started atomic.Bool
	stopCh  chan struct{}
}

func NewLocalSyncThrottle[K comparable](shardFn ...func(K) uint8) *LocalSyncThrottle[K] {
	t := &LocalSyncThrottle[K]{stopCh: make(chan struct{})}
	if len(shardFn) > 0 && shardFn[0] != nil {
		t.shardFn = shardFn[0]
	} else {
		seed := maphash.MakeSeed()
		t.shardFn = func(key K) uint8 {
			var h maphash.Hash
			h.SetSeed(seed)
			fmt.Fprint(&h, key)
			return uint8(h.Sum64())
		}
	}
	for i := range t.shards {
		t.shards[i].m = make(map[K]*_syncEntry)
	}
	t.StartJanitor(context.Background(), 30*time.Second, 5*time.Minute)
	return t
}

// Allow blocks until the key can run, then returns an unlock function.
func (t *LocalSyncThrottle[K]) Allow(key K) func() {
	s := &t.shards[t.shardFn(key)]

	s.mu.Lock()
	e := s.m[key]
	if e == nil {
		e = &_syncEntry{lastUnixNano: time.Now().UnixNano()}
		e.cond = sync.NewCond(&e.mu)
		s.m[key] = e
	}
	e.mu.Lock()
	s.mu.Unlock()

	ticket := e.next
	e.next++
	for e.serving != ticket {
		e.cond.Wait()
	}
	e.mu.Unlock()
	// 返回一个 release 释放锁的方法  用于在defer里调用
	return func() {
		e.mu.Lock()
		e.serving++
		e.lastUnixNano = time.Now().UnixNano()
		e.cond.Broadcast()
		e.mu.Unlock()
	}
}

// StartJanitor starts a background cleanup loop.
// interval: how often to scan
// gcAfter: delete keys not touched for longer than this duration
// 超时释放
func (t *LocalSyncThrottle[K]) StartJanitor(ctx context.Context, interval, gcAfter time.Duration) {
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
						e.mu.Lock()
						inUse := e.next != e.serving
						last := e.lastUnixNano
						e.mu.Unlock()
						if inUse {
							continue
						}
						if now-last > gcAfterNs {
							delete(s.m, k)
						}
					}
					s.mu.Unlock()
				}
			}
		}
	}()
}

func (t *LocalSyncThrottle[K]) Stop() {
	select {
	case <-t.stopCh:
		// already closed
	default:
		close(t.stopCh)
	}
}
