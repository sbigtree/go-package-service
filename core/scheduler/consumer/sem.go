package consumer

// ConcurrencyLimit 基于 chan 实现的并发控制器
type ConcurrencySem struct {
	tokenChan chan struct{} // 令牌桶
	tag       string        // 标识，用于日志
}

// NewConcurrencyLimit 创建并发控制器
// maxConcurrent: 最大并发数（令牌桶大小）
// tag: 标识（用于日志）
func NewConcurrencySem(maxConcurrent int, tag string) *ConcurrencySem {
	return &ConcurrencySem{
		tokenChan: make(chan struct{}, maxConcurrent),
		tag:       tag,
	}
}

// Acquire 获取并发令牌（无超时，必等待）
func (c *ConcurrencySem) Acquire() {
	c.tokenChan <- struct{}{} // 令牌桶满时阻塞，直到有令牌释放
}

// Release 释放并发令牌
func (c *ConcurrencySem) Release() {
	<-c.tokenChan // 归还令牌
}

// Current 获取当前已使用的并发数
func (c *ConcurrencySem) Current() int {
	return len(c.tokenChan)
}

// Max 获取最大并发数
func (c *ConcurrencySem) Max() int {
	return cap(c.tokenChan)
}
