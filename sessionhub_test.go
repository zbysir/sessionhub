package sessionhub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestWaitOne(t *testing.T) {
	eb := New(WithClearTime(60))

	Send := func() (eventId int64) {
		eventId = eb.Begin()

		go func() {
			time.Sleep(10 * time.Second)
			err := eb.Complete(eventId, eventId*100, nil)
			if err != nil {
				t.Fatalf("%v id: %d", err, eventId)
			}
		}()

		return eventId
	}

	Wait := func(eventId int64) (interface{}, error) {
		ctx, c := context.WithTimeout(context.TODO(), 15*time.Second)
		defer c()
		return eb.Wait(ctx, eventId)
	}

	eventId := Send()

	v, err := Wait(eventId)
	if err != nil {
		t.Fatal(err)
	}
	// 10s后会打印100
	t.Logf("1: %v", v)

	return

}

// 测试并发
func TestConcurrent(t *testing.T) {
	eb := New(WithClearTime(60))

	Send := func() (eventId int64) {
		eventId = eb.Begin()

		go func() {
			time.Sleep(10 * time.Second)
			err := eb.Complete(eventId, eventId*100, nil)
			if err != nil {
				t.Fatalf("%v id: %d", err, eventId)
			}
		}()

		return eventId
	}

	Wait := func(eventId int64) (interface{}, error) {
		ctx, c := context.WithTimeout(context.TODO(), 15*time.Second)
		defer c()
		return eb.Wait(ctx, eventId)
	}

	for range time.Tick(1 * time.Second) {
		for i := 0; i < 50; i++ {
			go func() {
				eventId := Send()
				v, err := Wait(eventId)
				if err != nil {
					t.Fatal(err)
				}

				t.Logf("%v", v)
			}()
		}
	}

	return

}

func TestWaitTwice(t *testing.T) {
	eb := New(WithClearTime(60))

	eventId := eb.Begin()

	go func() {
		time.Sleep(10 * time.Second)
		err := eb.Complete(eventId, eventId*100, nil)
		if err != nil {
			t.Fatalf("%v id: %d", err, eventId)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		v, e := eb.Wait(context.Background(), eventId)
		t.Logf("1: %v %v", v, e)
	}()
	go func() {
		defer wg.Done()
		v, e := eb.Wait(context.Background(), eventId)
		t.Logf("2: %v %v", v, e)
	}()

	wg.Wait()

	// 期待行为: 两次Wait只会有一次成功, 失败的一次报错: session closed
}

func TestChan(t *testing.T) {
	c := make(chan bool, 1)

	c <- true

	close(c)

	t.Log(<-c)
	t.Log(<-c)

}
