# Simple worker pool

Worker provides a simple worker pool to manage go routines

# Quick start

*Start a worker and put objects on a queue*
```go
import github.com/crosseyed/worker

w := worker.New(&worker.Options{
    Workers: 6
})

// Create a worker function
workerFunc := func(data interface{}){
    num, ok := data.(int)
    if !ok {
        fmt.Print("Not an integer type")
        return
    }

    fmt.Printf("Worker received value %d\n",  num)
}

// Start pool
w.Start(workerFunc)

// Put an object in the worker pool
for i := 0; i < 30; i++ {
    w.Put(i)
}

// Close communication and wait for all workers to complete
w.Close()
```

*Start a worker and process in batches*
```go
import github.com/crosseyed/worker

// Start 3 workers, each with a batch size of 24
w := worker.New(&worker.Options{
    Workers: 3
    BatchSize: 24
})

// Create a worker function which processes batches (arrays)
workerFunc := func(batch []interface{}){
    for _, val := range batch {
        num, ok := val.(int)
        if !ok {
            fmt.Print("Not an integer type")
            return
        }
        fmt.Printf("Worker received value %d\n",  num)
    }
}

// Start pool
w.Start(workerFunc)

// Put an object in the worker pool
for i := 0; i < 72; i++ {
    w.Put(i)
}

// Close communication and wait for all workers to complete
w.Close()
```
