package worker

import (
	"context"
	"sync"
)

var (
	// Put context
	putCtx = context.Background()
)

// Func is a type accepted by Start()
type Func func(interface{})

// FuncBatch is a type accepted by Start()
type FuncBatch func([]interface{})

// FuncContext is a contextulized function accepted by Start()
type FuncContext func(context.Context, interface{})

// FuncBatchContext is a contextulized function accepted by Start()
type FuncBatchContext func(context.Context, []interface{})

// Queue exposes queue interfaces
type Queue interface {
	Put(interface{}) error
	Flush()
}

// Manager exposes interfaces to manage worker lifecycle
type Manager interface {
	Start(interface{}) error
	Close()
	Halt()
}

// Sync synchronizes go routines
type Sync interface {
	Wait()
}

// Payload with a context
type Payload struct {
	Data    interface{}
	Context context.Context
}

// Options are options to pass to New()
type Options struct {
	// Number of Go routines
	Workers uint
	// Length of array to pass to WorkerBatchFunc
	BatchSize uint
	// Channel length
	ChanSize int
}

// Worker manages worker pools. It provides a framework to start go routine pools, push objects unto a queue,
// wait for channels to drain, wait for routines to exit and flush any objects that have been batched.
type Worker struct {
	Manager
	Queue
	Sync
	putMu     *sync.Mutex        // Synchronize Put, Flush & Close routines
	flushMu   *sync.Mutex        // Ensures the Flush call is thread safe
	flushWG   *sync.WaitGroup    // WaitGroup to determine when all routines have flushed all jobs that have been queued
	wrkrWG    *sync.WaitGroup    // WaitGroup to notify when all workers have exited upon channel close
	chanWG    *sync.WaitGroup    // WaitGroup to notify when channel has drained
	startOnce *sync.Once         // Start routine once
	closeOnce *sync.Once         // Close channel once
	closed    bool               // Channel is closed if true
	halt      bool               // Force the program not to process any more items
	batchSize uint               // Size of batch to buffer before sending to WorkerBatchFunc or WorkerFunc
	wrks      uint               // Number of concurrent routines to run
	ch        chan Payload       // Communication channel. The array length is controlled by batchSize
	chFlush   []chan interface{} // One channel per go routine to notify each routine to flush its batch objects
}

// New starts a *Worker object.
//
// See worker.Options for a list of options.
// Defaults: Workers = 1, ChanSize = 1
func New(o *Options) *Worker {
	var (
		batchsize uint
		workers   uint
		chanlen   int
	)
	if o != nil {
		batchsize = o.BatchSize
		workers = o.Workers
		chanlen = o.ChanSize
	}
	if chanlen < 1 {
		chanlen = 1
	}
	if workers < 1 {
		workers = 1
	}
	w := &Worker{
		putMu:     &sync.Mutex{},
		flushMu:   &sync.Mutex{},
		flushWG:   &sync.WaitGroup{},
		wrkrWG:    &sync.WaitGroup{},
		chanWG:    &sync.WaitGroup{},
		startOnce: &sync.Once{},
		closeOnce: &sync.Once{},
		batchSize: batchsize,
		wrks:      workers,
		ch:        make(chan Payload, chanlen),
		chFlush:   []chan interface{}{},
	}
	return w
}

// castFuncContext checks that fn is type Func or FuncContext otherwise return an error.
func castFuncContext(fn interface{}) (FuncContext, error) {
	switch f := fn.(type) {
	case Func:
		return func(_ context.Context, item interface{}) {
			f(item)
		}, nil
	case func(interface{}):
		return func(_ context.Context, item interface{}) {
			f(item)
		}, nil
	case FuncContext:
		return f, nil
	case func(context.Context, interface{}):
		return f, nil
	}
	return nil, &Error{worker: "Unknown type"}
}

// castFuncBatchContext checks that fn is type FuncBatch or FuncBatchContext
func castFuncBatchContext(fn interface{}) (FuncBatchContext, error) {
	switch f := fn.(type) {
	case FuncBatch:
		return func(_ context.Context, buf []interface{}) {
			f(buf)
		}, nil
	case func([]interface{}):
		return func(_ context.Context, buf []interface{}) {
			f(buf)
		}, nil
	case FuncBatchContext:
		return f, nil
	case func(context.Context, []interface{}):
		return f, nil
	}
	return nil, &Error{worker: "Unknown type"}
}

// Start starts the worker pool.
// Functions must follow the same signature as WorkerFunc or WorkerBatchFunc as the callback.
func (w *Worker) Start(fn interface{}) error {
	var f interface{}
	f1, err := castFuncContext(fn)
	f2, err := castFuncBatchContext(fn)
	switch {
	case f1 != nil:
		f = f1
	case f2 != nil:
		f = f2
	default:
		return err
	}

	wrks := uint(1)
	for i := uint(0); i < wrks; i++ {
		flush := make(chan interface{}, 2)
		w.chFlush = append(w.chFlush, flush)
		go start(w, flush, f)
	}
	return nil
}

func start(w *Worker, flush <-chan interface{}, action interface{}) {
	switch f := action.(type) {
	case FuncContext:
		startFuncContext(w, flush, f)
	case FuncBatchContext:
		startFuncBatchContext(w, flush, f)
	default:
		panic("Unknown Function Type: Should be FuncContext or FuncBatchContext")
	}
}

// startFuncContext starts a worker pool
func startFuncContext(w *Worker, flush <-chan interface{}, action FuncContext) {
	w.wrkrWG.Add(1)
	defer w.wrkrWG.Done()
	batch := map[context.Context][]interface{}{}
	for {
		select {
		case input, ok := <-w.ch:
			switch {
			case w.halt:
			case !ok:
				// On channel close
				for ctx, queued := range batch {
					if len(queued) > 0 {
						for _, item := range queued {
							action(ctx, item)
						}
						delete(batch, ctx)
					}
				}
				return
			case w.batchSize > 1:
				// Process in batches
				data := input.Data
				if _, ok := batch[input.Context]; !ok {
					batch[input.Context] = []interface{}{}
				}
				batch[input.Context] = append(batch[input.Context], data)
				if uint(len(batch[input.Context])) >= w.batchSize {
					for _, item := range batch[input.Context] {
						action(input.Context, item)
					}
					delete(batch, input.Context)
				}
			default:
				// Process individually
				action(input.Context, input.Data)
			}
			w.chanWG.Done()
		case <-flush:
			for ctx, queued := range batch {
				if len(queued) > 0 {
					for _, item := range queued {
						action(ctx, item)
					}
					batch[ctx] = []interface{}{}
				}
			}
			w.flushWG.Done()
			w.flushWG.Wait()
		}
	}
}

// startFuncBatchContext starts a worker pool
func startFuncBatchContext(w *Worker, flush <-chan interface{}, action FuncBatchContext) {
	w.wrkrWG.Add(1)
	defer w.wrkrWG.Done()
	batch := map[context.Context][]interface{}{}
	for {
		select {
		case input, ok := <-w.ch:
			switch {
			case w.halt:
			case !ok:
				// On channel close
				for ctx, queued := range batch {
					if len(queued) > 0 {
						action(ctx, queued)
						delete(batch, ctx)
					}
				}
				return
			case w.batchSize > 1:
				// Process in batches
				ctx := input.Context
				data := input.Data
				if _, ok := batch[ctx]; !ok {
					batch[ctx] = []interface{}{}
				}
				batch[ctx] = append(batch[ctx], data)
				if uint(len(batch[ctx])) >= w.batchSize {
					action(ctx, batch[ctx])
					batch[ctx] = []interface{}{}
				}
			default:
				// Process individually
				ctx := input.Context
				data := input.Data
				action(ctx, []interface{}{data})
			}
			w.chanWG.Done()
		case <-flush:
			for ctx, queued := range batch {
				if len(queued) > 0 {
					action(ctx, queued)
					batch[ctx] = []interface{}{}
				}
			}
			w.flushWG.Done()
			w.flushWG.Wait()
		}
	}
}

// Put adds a job unto the queue
func (w *Worker) Put(input interface{}) error {
	return w.PutC(putCtx, input)
}

// PutC adds a contextualised job to the queue
func (w *Worker) PutC(ctx context.Context, input interface{}) error {
	w.putMu.Lock()
	defer w.putMu.Unlock()
	if w.closed {
		return &Error{worker: "input queue is closed"}
	}
	if w.halt {
		return &Error{worker: "process has been killed"}
	}
	payload := Payload{
		Context: ctx,
		Data:    input,
	}
	w.chanWG.Add(1)
	w.ch <- payload
	return nil
}

// Flush informs wrks to flush any batches that have been queued. Blocks until all routines have flushed.
func (w *Worker) Flush() {
	w.flushMu.Lock()
	w.putMu.Lock()
	defer func() {
		w.flushMu.Unlock()
		w.putMu.Unlock()
	}()
	w.chanWG.Wait()
	w.flushWG.Add(len(w.chFlush))
	for _, ch := range w.chFlush {
		ch <- nil
	}
	w.flushWG.Wait()
}

// Close closes the input queue and blocks until the queue is completely processed
func (w *Worker) Close() {
	w.closeOnce.Do(func() {
		w.putMu.Lock()
		defer w.putMu.Unlock()
		close(w.ch)
		w.Wait()
		w.closed = true
	})
}

// Halt will force all routines to stop without processing the queued or batched jobs
func (w *Worker) Halt() {
	w.putMu.Lock()
	defer w.putMu.Unlock()
	w.halt = true
}

// Wait waits for all go routines to shutdown. Shutdown is triggered by calling Close
func (w *Worker) Wait() {
	w.flushWG.Wait()
	w.chanWG.Wait()
	w.wrkrWG.Wait()
}
