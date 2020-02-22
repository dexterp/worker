package worker

import "fmt"

func ExampleNew() {
	maxcount := 30
	var totalcount int

	w := New(&Options{
		Workers: 6,
	})

	fn := func(data interface{}) {
		d, ok := data.(int)
		if !ok {
			fmt.Printf("data is not an integer")
			return
		}
		totalcount += d
	}

	w.Start(fn)

	for i := 0; i < maxcount; i++ {
		w.Put(1)
	}
	w.Close()

	if maxcount == totalcount {
		fmt.Print("Counts match")
	} else {
		fmt.Print("Counts do not match")
	}
}
