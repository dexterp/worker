package message

import (
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMessage_Send_1(t *testing.T) {
	var (
		min  int = 10
		max  int = 30
		send int
		recv int
	)
	m := New(nil)
	go func() {
		v, err := m.Recv()
		if err != nil {
			t.Errorf("Recieved an error: %v", err)
		}
		val, ok := v.(int)
		if !ok {
			t.Error("Expected an integer type")
		}
		recv = val
	}()

	send = rand.Intn(max-min) + min
	m.Send(send)
	m.Wait()

	if send != recv {
		t.Logf("%d != %d", send, recv)
		t.Fail()
	}
}

func TestMessage_Send_2(t *testing.T) {
	var (
		min  int = 10
		max  int = 30
		send int
		recv int
	)
	m := New(&Options{
		ChanSize: 0,
	})
	mu := &sync.Mutex{}
	go func() {
		mu.Lock()
		defer mu.Unlock()
		v, err := m.Recv()
		if err != nil {
			t.Errorf("Recieved an error: %v", err)
		}
		val, ok := v.(int)
		if !ok {
			t.Error("Expected an integer type")
		}
		recv = val
	}()

	send = rand.Intn(max-min) + min
	m.Send(send)
	m.Wait()

	if send != recv {
		t.Logf("%d != %d", send, recv)
		t.Fail()
	}
}

func TestClose_1(t *testing.T) {
	m := New(&Options{ChanSize: 1})

	err := m.Send(1)
	if err != nil {
		t.Fail()
	}

	_, err = m.Recv(true)
	if err != nil {
		t.Fail()
	}
	m.Close()

	err = m.Send(2)
	if err == nil {
		t.Error("Expecting error when channel is closed")
	} else if !strings.Contains(err.Error(), "channel closed") {
		t.Error("Error is not channel closed")
	}
}

func TestClose_2(t *testing.T) {
	m := New(&Options{ChanSize: 1})

	err := m.Send(1)
	if err != nil {
		t.Fail()
	}

	_, err = m.Recv(true)
	if err != nil {
		t.Fail()
	}
	m.Close()

	fn := func() <-chan interface{} {
		ch := make(chan interface{}, 1)
		go func() {
			_, err = m.Recv(true)
			if err == nil {
				t.Error("Expecting error when channel is closed")
			} else if !strings.Contains(err.Error(), "channel closed") {
				t.Error("Error is not channel closed")
			}
			ch <- "done"
		}()
		return ch
	}

	select {
	case <-fn():
	case <-time.After(30 * time.Second):
		t.Error("Time out running test")
	}
}
