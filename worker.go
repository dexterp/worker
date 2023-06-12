package worker

import (
	"context"
	"sync"
	"sync/atomic"
)

var (
	// Put context
	putCtx = context.Background()
)

// Func is a type accepted by Start()
type Func[T any] func(T)

// FuncBatch is a type accepted by Start()
type FuncBatch[T any] func([]T)

// FuncContext is a contextualized function accepted by Start()
type FuncContext[T any] func(context.Context, T)

// FuncBatchContext is a contextualized function accepted by Start()
type FuncBatchContext[T any] func(context.Context, []T)

// Queue exposes queue interfaces
type Queue[T any] interface {
	Put(T) error
	Flush()
}

// Manager exposes interfaces to manage worker lifecycle
type Manager[T any] interface {
	Start(T) error
	Close()
	Halt()
}

// Sync synchronizes go routines
type Sync interface {
	Wait()
}

// Payload with a context
type Payload[T any] struct {
	Data    T
	Context context.Context
}

// Options are options to pass to New()
type Options struct {
	// Number of Go routines
	Workers uint
	// Length of slice to pass to WorkerBatchFunc
	BatchSize uint
	// Channel size
	ChanSize int
}

// Worker manages worker pools. It provides a framework to start go routine pools, push objects unto a queue,
// wait for channels to drain, wait for routines to exit and flush any objects that have been batched.
type Worker[T any] struct {
	Manager[T]
	Queue[T]
	Sync
	putMu      *sync.Mutex     // Synchronize Put, Flush & Close routines.
	flushMu    *sync.Mutex     // Ensures the Flush call can only run one at a time.
	flushWG    *sync.WaitGroup // WaitGroup to determine when all routines have flushed all queued jobs.
	wrkrWG     *sync.WaitGroup // WaitGroup to notify when all workers have exited upon channel close.
	chanWG     *sync.WaitGroup // WaitGroup to notify when channel has drained.
	startOnce  *sync.Once      // Start routine once.
	closeOnce  *sync.Once      // Close channel once.
	haltAtomic *atomic.Bool    // Force the program not to process any more items.
	closed     bool            // Channel is closed when true.
	batchSize  uint            // Size of batch to buffer before sending to WorkerBatchFunc or WorkerFunc.
	wrks       uint            // Number of concurrent routines to run.
	ch         chan Payload[T] // Communication channel. The slice length is controlled by batchSize.
	chFlush    []chan struct{} // One channel per go routine to notify each routine to flush its batch objects.
}

// New starts a *Worker object.
//
// See worker.Options for a list of options.
// Defaults: Workers = 1, ChanSize = 1
func New[T any](option ...Options) *Worker[T] {
	var o *Options
	if len(option) > 0 {
		o = &option[0]
	}
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
	w := &Worker[T]{
		putMu:      &sync.Mutex{},
		flushMu:    &sync.Mutex{},
		flushWG:    &sync.WaitGroup{},
		wrkrWG:     &sync.WaitGroup{},
		chanWG:     &sync.WaitGroup{},
		startOnce:  &sync.Once{},
		closeOnce:  &sync.Once{},
		haltAtomic: &atomic.Bool{},
		batchSize:  batchsize,
		wrks:       workers,
		ch:         make(chan Payload[T], chanlen),
		chFlush:    []chan struct{}{},
	}
	return w
}

// castFuncContext verifies the function signature of "fn" matches the Func or FuncContext signature.
func castFuncContext[T any](fn interface{}) (FuncContext[T], error) {
	switch f := fn.(type) {
	case Func[T]:
		return func(_ context.Context, item T) {
			f(item)
		}, nil
	case FuncContext[T]:
		return f, nil
	case func(T):
		return func(_ context.Context, item T) {
			f(item)
		}, nil
	case func(context.Context, T):
		return f, nil
	}
	return nil, &Error{msg: "unknown function type"}
}

// castFuncBatchContext verifies the function signature of "fn" matches the FuncBatch or FuncBatchContext signature.
func castFuncBatchContext[T any](fn interface{}) (FuncBatchContext[T], error) {
	switch f := fn.(type) {
	case FuncBatch[T]:
		return func(_ context.Context, buf []T) {
			f(buf)
		}, nil
	case FuncBatchContext[T]:
		return f, nil
	case func([]T):
		return func(_ context.Context, buf []T) {
			f(buf)
		}, nil
	case func(context.Context, []T):
		return f, nil
	}
	return nil, &Error{msg: "unknown function type"}
}

// Start starts the worker pool. The function signature must match the signature
// of WorkerFunc or WorkerBatchFunc.
func (w *Worker[T]) Start(fn interface{}) error {
	var f interface{}
	f1, err := castFuncContext[T](fn)
	f2, err := castFuncBatchContext[T](fn)
	switch {
	case f1 != nil:
		f = f1
	case f2 != nil:
		f = f2
	default:
		return err
	}

	for i := uint(0); i < w.wrks; i++ {
		flush := make(chan struct{}, 2)
		w.chFlush = append(w.chFlush, flush)
		w.wrkrWG.Add(1)
		go start(w, flush, f)
	}
	return nil
}

// start initiates a worker or a batch worker.
func start[T any](w *Worker[T], flush <-chan struct{}, action interface{}) {
	switch f := action.(type) {
	case FuncContext[T]:
		startFuncContext[T](w, flush, f)
	case FuncBatchContext[T]:
		startFuncBatchContext[T](w, flush, f)
	default:
		panic("unknown function type: should be FuncContext or FuncBatchContext")
	}
}

// startFuncContext initiates a worker.
func startFuncContext[T any](w *Worker[T], flush <-chan struct{}, action FuncContext[T]) {
	defer w.wrkrWG.Done()
	batch := map[context.Context][]T{}
	for {
		select {
		case input, ok := <-w.ch:
			switch {
			case w.haltAtomic.Load():
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
					batch[input.Context] = []T{}
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
					batch[ctx] = []T{}
				}
			}
			w.flushWG.Done()
			w.flushWG.Wait()
		}
	}
}

// startFuncBatchContext initiates a batch worker.
func startFuncBatchContext[T any](w *Worker[T], flush <-chan struct{}, action FuncBatchContext[T]) {
	defer w.wrkrWG.Done()
	batch := map[context.Context][]T{}
	for {
		select {
		case input, ok := <-w.ch:
			switch {
			case w.haltAtomic.Load():
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
					batch[ctx] = []T{}
				}
				batch[ctx] = append(batch[ctx], data)
				if uint(len(batch[ctx])) >= w.batchSize {
					action(ctx, batch[ctx])
					batch[ctx] = []T{}
				}
			default:
				// Process individually
				ctx := input.Context
				data := input.Data
				action(ctx, []T{data})
			}
			w.chanWG.Done()
		case <-flush:
			for ctx, queued := range batch {
				if len(queued) > 0 {
					action(ctx, queued)
					batch[ctx] = []T{}
				}
			}
			w.flushWG.Done()
			w.flushWG.Wait()
		}
	}
}

// Put adds a new job to the queue.
func (w *Worker[T]) Put(input T) error {
	return w.PutC(putCtx, input)
}

// PutC inserts a contextualised job onto the queue.
func (w *Worker[T]) PutC(ctx context.Context, input T) error {
	w.putMu.Lock()
	defer w.putMu.Unlock()
	if w.closed {
		return &Error{msg: "the channel has been closed"}
	}
	if w.haltAtomic.Load() {
		return &Error{msg: "worker pool has been halted"}
	}
	payload := Payload[T]{
		Context: ctx,
		Data:    input,
	}
	w.chanWG.Add(1)
	w.ch <- payload
	return nil
}

// Flush notifies workers to flush any queued batches. Blocks until all Goroutines have finished flushing.
func (w *Worker[T]) Flush() {
	w.flushMu.Lock()
	w.putMu.Lock()
	defer func() {
		w.flushMu.Unlock()
		w.putMu.Unlock()
	}()
	w.chanWG.Wait()
	w.flushWG.Add(len(w.chFlush))
	for _, ch := range w.chFlush {
		ch <- struct{}{}
	}
	w.flushWG.Wait()
}

// Close closes the input queue and waits for the items in the queue to be processed.
func (w *Worker[T]) Close() {
	w.closeOnce.Do(func() {
		w.putMu.Lock()
		defer w.putMu.Unlock()
		close(w.ch)
		w.Wait()
		w.closed = true
	})
}

// Halt force all routines to stop without processing any queued or batched jobs.
func (w *Worker[T]) Halt() {
	w.putMu.Lock()
	defer w.putMu.Unlock()
	w.haltAtomic.Store(true)
}

// Wait waiting for all go routines to exit. The shutdown is initiated by
// calling the Close function.
func (w *Worker[T]) Wait() {
	w.flushWG.Wait()
	w.chanWG.Wait()
	w.wrkrWG.Wait()
}

// Start start a worker pool with function fn. Verifies the function signature
// of fn matches the Func, FuncContext, FuncBatch or FuncBatchContext
// signature.
func Start[T any](fn any, options ...Options) (*Worker[T], error) {
	w := New[T](options...)
	err := w.Start(fn)
	if err != nil {
		return nil, err
	}
	return w, nil
}
