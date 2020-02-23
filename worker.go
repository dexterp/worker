package worker

import (
	"sync"
)

// Func is a type accepted by Start()
type Func func(interface{})

// FuncBatch is a type accepted by Start()
type FuncBatch func([]interface{})

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
	ch        chan []interface{} // Communication channel. The array length is controlled by batchSize
	chFlush   []chan interface{} // One channel per go routine to notify each routine to flush its batch objects
}

// New starts a *Worker object.
//
// See worker.Options for a list of options.
// Defaults:
// - Workers = 1
// - ChanSize = 1
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
		ch:        make(chan []interface{}, chanlen),
		chFlush:   []chan interface{}{},
	}
	return w
}

// Start starts the worker pool.
// Functions must follow the same signature as WorkerFunc or WorkerBatchFunc as the callback.
func (w *Worker) Start(fn interface{}) error {
	f, err := assertFunc(fn)
	if err != nil {
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

// assertFunc checks that fn is type Func or FuncBatch otherwise return an error.
// If function type is Func it will be wrapped within the returning FuncBatch
func assertFunc(fn interface{}) (FuncBatch, error) {
	switch f := fn.(type) {
	case Func:
		return func(buf []interface{}) {
			for _, b := range buf {
				f(b)
			}
		}, nil
	case func(interface{}):
		return func(buf []interface{}) {
			for _, b := range buf {
				f(b)
			}
		}, nil
	case FuncBatch:
		return f, nil
	case func([]interface{}):
		return f, nil
	}
	return nil, &Error{worker: "Unknown type"}
}

func start(w *Worker, flush <-chan interface{}, action FuncBatch) {
	w.wrkrWG.Add(1)
	defer w.wrkrWG.Done()
	var batch []interface{}
	for {
		select {
		case input, ok := <-w.ch:
			switch {
			case w.halt:
			case !ok:
				// On channel close
				if len(batch) > 0 {
					action(batch)
					batch = nil
				}
				return
			case w.batchSize > 1:
				// Process in batches
				for _, item := range input {
					batch = append(batch, item)
					if uint(len(batch)) >= w.batchSize {
						action(batch)
						batch = nil
					}
				}
			default:
				// Process individually
				for _, item := range input {
					action([]interface{}{item})
				}
			}
			c := len(input)
			if c > 0 {
				w.chanWG.Add(-c)
			}
		case <-flush:
			if len(batch) > 0 {
				action(batch)
				batch = nil
			}
			w.flushWG.Done()
			w.flushWG.Wait()
		}
	}
}

// Put adds a list of jobs to the queue.
func (w *Worker) Put(input interface{}) error {
	w.putMu.Lock()
	defer w.putMu.Unlock()
	if w.closed {
		return &Error{worker: "input queue is closed"}
	}
	if w.halt {
		return &Error{worker: "process has been killed"}
	}
	var payload []interface{}
	if list, ok := input.([]interface{}); ok {
		payload = list
	} else {
		payload = []interface{}{input}
	}
	l := len(payload)
	w.chanWG.Add(l)
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
	w.wrkrWG.Wait()
}
