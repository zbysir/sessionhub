package sessionhub

import "sync/atomic"

type option struct {
	// 如果Begin之后太久没有Commit或者Wait, 那么这个监听将自动删除以保证不会内存泄露
	// 设置的值应该大于Wait超时时间.
	clearTime   int64
	idGenerator IdGenerator
}

type Option func(*option)

func WithClearTime(t int64) Option {
	return func(o *option) {
		o.clearTime = t
	}
}

func WithIdGenerator(g IdGenerator) Option {
	return func(o *option) {
		o.idGenerator = g
	}
}

// id生成器
type IdAdd struct {
	x int64
}

func (i *IdAdd) NextId() int64 {
	return atomic.AddInt64(&i.x, 1)
}
