# SessionHub

开启一个会话, 并等待它完成.

用于在长连接中 如Grpc Stream, WebSocket, 服务器或客户端发送一个消息并等待对方回应, 实现在一个异步连接中同步等待回话.

## Usage

```
package main

import (
	"context"
	"fmt"
	"go.zhuzi.me/go/sessionhub"
	"time"
)

var wo = sessionhub.New()

func main() {
	sessionId := wo.Begin()

	go func() {
		err := sendMsgToClient(sessionId)
		if err != nil {
			wo.Close(sessionId)
			return
		}
	}()

	value, err := wo.Wait(context.TODO(), sessionId)

	fmt.Printf("%v %v", value, err)
}

func sendMsgToClient(sessionId int64) (err error) {
	// 通过网络发送消息...

	// 模拟等待一秒后客户端响应消息
	go func() {
		time.Sleep(1 * time.Second)

		// 实际上, sessionId应该通过网络传输, 这里只是模拟
		onMsg(sessionId)
	}()

	return nil
}

func onMsg(session int64) {
	wo.Complete(session, 1, nil)
}

```

## How does it work?

和http2中的stream identifier类似, 每个消息体都需要包含sessionId来识别消息属于哪个会话, 由SessionHub通知正在等待中的会话.

