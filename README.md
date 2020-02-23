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
    if num, ok := data.(int); !ok {
        fmt.Print("Not an integer type")
        return
    }

    fmt.Printf("Worker received value %d",  num)
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