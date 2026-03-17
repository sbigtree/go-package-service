package scheduler

import (
	"fmt"
	"sync"
)

type GoroutinePool struct {
	sem chan struct{}
	wg  sync.WaitGroup
}

func NewGoroutinePool(size int) *GoroutinePool {
	if size < 0 {
		size = 10
	}

	return &GoroutinePool{
		sem: make(chan struct{}, size),
	}
}

func (p *GoroutinePool) Go(fn func()) {
	p.wg.Add(1)
	p.sem <- struct{}{}
	go func() {
		defer func() {
			defer p.wg.Done()
			<-p.sem
			if r := recover(); r != nil {
				fmt.Println("groutine panic:", r)
			}
		}()
		fn()
	}()
}

func (p *GoroutinePool) Wait() {
	p.wg.Wait()
}

// 禁止动态扩容（生产安全）
func (p *GoroutinePool) Resize(size int) {
	fmt.Println("⚠️ 不建议动态Resize，会导致并发失控")
}
