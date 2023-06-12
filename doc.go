/*
Package worker provides functions to create and manage worker pool.
It supports context workers and batch queuing

worker simplifies creation of pools and queuing jobs.

	// Create a worker function
	workerFunc := func(num int){
		fmt.Printf("Worker received value %d\n",  num)
	}

	// Start pool
	w, err := worker.Start[int](workerFunc)
	if err != nil {
		// handle error
	}

	// Put an object in the worker pool
	for i := 0; i < 30; i++ {
		w.Put(i)
	}

	// Close communication and wait for all workers to complete
	w.Close()

worker supports batch queuing.

	w := worker.New(&worker.Options{
		Workers:   2,
		BatchSize: 16,
	})

	// Worker function
	workerFunc := func(batch []int) {
		for _, num := range batch {
			sum += num
		}
		// Prints 16
		fmt.Printf("Batch Count %d\n", count)
	}

	// Start workers
	w, err := worker.Start[int](workerFunc, worker.Options{
		Workers:   2,
		BatchSize: 16,
	})
	if err != nil {
		// handle error
	}

	// Put an object in the worker pool
	for i := 0; i < 32; i++ {
		w.Put(10)
	}
*/
package worker
