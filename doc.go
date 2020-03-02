/*
Package worker creates worker pools.

worker simplifies creation of pools and queuing jobs.

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

worker supports batch queuing.

	// Setup two worker with a batch size of sixteen
	w := New(&Options{
		Workers:   2,
		BatchSize: uint(16),
	})

	// Worker function
	workerFunc := func(data []interface{}) {
		var count int
		for _, item := range data {
			num, ok := data.(int)
			if !ok {
				fmt.Print("Not an integer type")
				return
			}
			count += 1
		}
		fmt.Printf("Batch Count %d\n", count)
	}

	// Start workers
	w.Start(workerFunc)

	// Put an object in the worker pool
	for i := 0; i < 32; i++ {
		w.Put(i)
	}

*/
package worker
