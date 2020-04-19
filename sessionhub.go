package sessionhub

import (
	"context"
	"errors"
	"sync"
	"time"
)

type msg struct {
	value interface{}
	err   error
}

type session struct {
	activeAt  int64
	sessionId int64
	c         chan msg

	m         sync.Mutex
	completed bool
	closed    bool
}

func (c *session) close() {
	c.m.Lock()
	defer c.m.Unlock()

	if c.closed {
		return
	}
	c.closed = true

	close(c.c)
}

func (c *session) complete(value interface{}, err error) error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.closed {
		return errors.New("session has closed")
	}

	if c.completed {
		return errors.New("session has completed")
	}
	c.completed = true

	c.c <- msg{
		value: value,
		err:   err,
	}

	close(c.c)
	c.closed = true

	return nil
}

type IdGenerator interface {
	NextId() int64
}

type SessionHub struct {
	// 所有事件, 可用最小堆优化clear时的性能
	sessions map[int64]*session
	// lock for sessions map
	mu sync.Mutex
	// id生成器, 每次+1, 如果SessionHub用在分布式的多个节点上那么id可能重复, 重复的id可能导致Commit和Wait不能一一对应. 如果需要可以修改为分布式唯一id.
	idGenerator IdGenerator
	clearTime   int64
}

func New(os ...Option) *SessionHub {
	var o option

	for _, v := range os {
		v(&o)
	}

	if o.clearTime == 0 {
		o.clearTime = 60
	}
	if o.idGenerator == nil {
		o.idGenerator = &IdAdd{}
	}

	e := &SessionHub{
		sessions:    map[int64]*session{},
		idGenerator: o.idGenerator,
		clearTime:   o.clearTime,
		mu:          sync.Mutex{},
	}
	go e.clear()

	return e
}

func (e *SessionHub) addSession(sessionId int64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.sessions[sessionId] = &session{
		activeAt:  time.Now().Unix(),
		sessionId: sessionId,
		c:         make(chan msg, 1),
	}
}

func (e *SessionHub) delSession(sessionId int64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	c, ok := e.sessions[sessionId]
	if ok {
		c.close()
		delete(e.sessions, sessionId)
	}
}

func (e *SessionHub) getSession(sessionId int64) (c *session, ok bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	c, ok = e.sessions[sessionId]
	if ok {
		c.activeAt = time.Now().Unix()
	}
	return
}

// Begin 开启一个会话, 返回会话id, 同一个会话id的多次Commit会被同一个Wait接受.
func (e *SessionHub) Begin() (sessionId int64) {
	sessionId = e.idGenerator.NextId()

	e.addSession(sessionId)
	return
}

// Close 关闭一个会话, 一般不需要调用, 当会话Wait成功或者超时失败后会自动删除.
// 当确认无法再继续Wait或者Commit的时候可以提前关闭以释放资源;
// 如begin之后 发送消息给对方失败, 这种情况下就无法再继续wait, 就应该手动关闭session.
func (e *SessionHub) Close(sessionId int64) {
	e.delSession(sessionId)
}

// Complete完成并传递结果给Wait, 每个sessionId只能Complete一次.
func (e *SessionHub) Complete(sessionId int64, value interface{}, err error) error {
	ev, ok := e.getSession(sessionId)
	if !ok {
		return errors.New("session not exist or cleared")
	}

	return ev.complete(value, err)
}

// Wait 等待事件Complete
// 同一个会话Id允许(但不推荐)被Wait多次, 但只会有一个成功.
func (e *SessionHub) Wait(ctx context.Context, sessionId int64) (value interface{}, err error) {
	ev, ok := e.getSession(sessionId)
	if !ok {
		err = errors.New("session not exist or closed")
		return
	}
	defer e.Close(sessionId)

	select {
	case msg, ok := <-ev.c:
		if ok {
			value = msg.value
			err = msg.err
		} else {
			// ev.c被关闭的可能原因:
			// 1. 太长时间没有Complete被clear
			// 2. wait两次, 第二次wait时session已经被第一次wait关闭了
			// 3. 用户手动close
			err = errors.New("session closed")
			return
		}

		return
	case <-ctx.Done():
		err = ctx.Err()

		return
	}
}

// 由于开发者可以只begin, 不commit, 所以会存在无效(超时)的事件.
// clear会清理掉无效的事件, 以保证不会内存泄露
func (e *SessionHub) clear() {
	for range time.Tick(60 * time.Second) {
		now := time.Now().Unix()
		e.mu.Lock()
		for _, ev := range e.sessions {
			if ev.activeAt < now-e.clearTime {

				ev.close()
				delete(e.sessions, ev.sessionId)

				continue
			}
		}

		e.mu.Unlock()
	}
}
