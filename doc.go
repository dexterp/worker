/*
Package worker provides functions to create and manage worker pool.
It supports context workers and batch queuing

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
	w := worker.New(&worker.Options{
		Workers:   2,
		BatchSize: 16,
	})

	// Worker function
	workerFunc := func(data []interface{}) {
		var sum int
		for _, item := range data {
			num, ok := data.(int)
			if !ok {
				fmt.Print("Not an integer type")
				return
			}
			sum += num
		}
		// Prints 16
		fmt.Printf("Batch Count %d\n", count)
	}

	// Start workers
	w.Start(workerFunc)

	// Put an object in the worker pool
	for i := 0; i < 32; i++ {
		w.Put(10)
	}

*/
package worker
