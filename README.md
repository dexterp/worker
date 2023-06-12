[![Github Actions](https://github.com/dexterp/worker/actions/workflows/go.yml/badge.svg)](https://github.com/dexterp/worker/actions) [![Go Report Card](https://goreportcard.com/badge/dexterp/worker)](https://goreportcard.com/report/dexterp/worker)  [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/dexterp/worker/blob/master/LICENSE)

# Simple worker pool

Worker provides a simple worker pool to manage go routines

# Quick start

*Start a worker and put objects on a queue*
```go
mu := &sync.Mutex{}
total := 0
// Create a worker function.
workerFunc := func(value int){
    mu.Lock()
    defer mu.Unlock()
    total += value
}

// Start pool.
w, err := worker.Start(workerFunc, worker.Options{
    Workers: 6,
})

// Put an object in the worker pool.
for i := 0; i < 30; i++ {
    w.Put(i)
}

// prints 30
fmt.Println(total)

// Close communication and wait for all workers to complete.
w.Close()
```

*Start a worker and process in batches*
```go
// Create a worker function which processes batches (slices).
workerFunc := func(batch []int){
    for _, val := range batch {
        // process in batches
    }
}

// Start pool.
w, err := worker.Start(workerFunc, worker.Options{
    Workers: 3,
    BatchSize: 24,
})

// Put an object in the worker pool.
for i := 0; i < 72; i++ {
    w.Put(i)
}

// Close communication and wait for all workers to complete.
w.Close()
```
