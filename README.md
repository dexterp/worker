# Simple worker pool

Worker provides a simple worker pool to manage go routines

# Quick start

```go
import github.com/crosseyed/worker

w := worker.New(&worker.Options{
    Workers: 6
})

fn := func(data interface{}){
    if num, ok := data.(int); !ok {
        fmt.Print("Not an integer type")
        return
    }

    fmt.Printf("Worker %d: received value %d", id.Num, num)
}

w.Start(fn)

for i := 0; i < 30; i++ {
    w.Put(i)
}
```