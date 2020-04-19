package main

import (
	"context"
	"fmt"
	"github.com/zbysir/sessionhub"
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
	// 通过网络发送消息
	// xxx

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
