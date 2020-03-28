package message

import (
	"context"
	"sync"
)

// Payload with a context
type payload struct {
	Data    interface{}
	Context context.Context
}

// Options provides options to New()
type Options struct {
	// Channel length
	ChanSize int
}

// New returns a Message
func New(o *Options) *Message {
	chlen := 1
	if o != nil {
		if o.ChanSize < 1 {
			chlen = o.ChanSize
		}
	}
	m := Message{
		sendMu:    &sync.Mutex{},
		chanWG:    &sync.WaitGroup{},
		ch:        make(chan *payload, chlen),
		closeOnce: &sync.Once{},
		closed:    false,
	}
	return &m
}

// Message passes messages between processes
type Message struct {
	sendMu    *sync.Mutex     // Synchronize Put, Flush & Close routines
	chanWG    *sync.WaitGroup // WaitGroup to notify when channel has drained
	ch        chan *payload   // channel payload
	closeOnce *sync.Once      // Close channel once
	closed    bool            // Channel is closed if true
}

// Send adds a job unto the queue
func (m *Message) Send(input interface{}) error {
	putCtx := context.Background()
	return m.SendC(putCtx, input)
}

// SendC adds a contextualised job to the queue
func (m *Message) SendC(ctx context.Context, input interface{}) error {
	m.sendMu.Lock()
	defer m.sendMu.Unlock()
	if m.closed {
		return &Error{Msg: "channel closed"}
	}
	pl := payload{
		Context: ctx,
		Data:    input,
	}
	m.chanWG.Add(1)
	m.ch <- &pl
	return nil
}

// Recv receives a message. If done is set it will reduce the waitgroup by one
func (m *Message) Recv(done ...interface{}) (interface{}, error) {
	_, pl, err := m.RecvC(done)
	return pl, err
}

// RecvC receives a contextual message
func (m *Message) RecvC(done ...interface{}) (context.Context, interface{}, error) {
	pl, ok := <-m.ch
	if !ok {
		return nil, nil, &Error{Msg: "channel closed"}
	}
	defer func() {
		if len(done) > 0 {
			m.chanWG.Done()
		}
	}()
	return pl.Context, pl.Data, nil
}

// Wait waits for all go routines to shutdown. Shutdown is triggered by calling Close
func (m *Message) Wait() {
	m.chanWG.Wait()
}

// Close closes the input queue and blocks until the queue is completely processed
func (m *Message) Close() {
	m.closeOnce.Do(func() {
		m.sendMu.Lock()
		defer m.sendMu.Unlock()
		close(m.ch)
		m.Wait()
		m.closed = true
	})
}
