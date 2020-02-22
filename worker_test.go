package worker

import (
	"math/rand"
	"sync"
	"testing"
)

func TestWorker_Start_1(t *testing.T) {
	fillcnt := 24
	var count int
	w := New(nil)
	fn := func(single interface{}) {
		if i, ok := single.(int); !ok {
			t.Error("Could not determine original value")
			return
		} else {
			count += i
		}
	}
	err := w.Start(fn)
	if err != nil {
		t.Errorf("%v", err)
	}
	for i := 0; i < fillcnt; i++ {
		err := w.Put(1)
		if err != nil {
			t.Errorf("Unexpected Put() error: %v", err)
		}
	}
	w.Close()
	if fillcnt != count {
		t.Errorf("Counts do not match %w = %w", fillcnt, count)
	}
}

func TestWorker_Start_2(t *testing.T) {
	w := New(nil)
	err := w.Start(func() {})
	if err == nil {
		t.Error("Expecting unknown type")
	}

}

func TestWorker_Put_1(t *testing.T) {
	var count int
	fn := func(single interface{}) {
		if i, ok := single.(int); !ok {
			t.Error("Expecting an int got")
			return
		} else {
			count += i
		}
	}
	w := New(nil)
	if err := w.Start(fn); err != nil {
		t.Errorf("Got an error: %v", err)
	}
	if err := w.Put(1); err != nil {
		t.Errorf("Got an error: %v", err)
	}
	w.Close()
	if err := w.Put(1); err == nil {
		t.Error("Test failed")
	}
}

func TestWorker_Put_2(t *testing.T) {
	var count int
	fn := func(single interface{}) {
		if i, ok := single.(int); !ok {
			t.Error("Expecting an int got")
			return
		} else {
			count += i
		}
	}
	w := New(nil)
	if err := w.Start(fn); err != nil {
		t.Errorf("Got an error: %v", err)
	}
	if err := w.Put(1); err != nil {
		t.Errorf("Got an error: %v", err)
	}
	w.Halt()
	if err := w.Put(1); err == nil {
		t.Error("Test failed")
	}
}

func TestWorker_Put_3(t *testing.T) {
	var total int
	fn := func(data []interface{}) {
		for _, d := range data {
			if val, ok := d.(int); ok {
				total += val
				continue
			}
			t.Error("Unknown type")
			return
		}
	}
	payload := []interface{}{20, 20, 20}
	w := New(nil)
	if err := w.Start(fn); err != nil {
		t.Errorf("Got an error: %v", err)
	}
	if err := w.Put(payload); err != nil {
		t.Errorf("Got an error: %v", err)
	}
}

func TestWorker_Func_1(t *testing.T) {
	min := 15
	max := 4096
	randnum := rand.Intn(max-min) + min
	var c int
	mu := &sync.Mutex{}
	fn := Func(func(single interface{}) {
		mu.Lock()
		defer mu.Unlock()
		if _, ok := single.(int); !ok {
			t.Error("Could not determine original value")
		}
		c += 1
	})
	w := New(&Options{
		Workers:   3,
		BatchSize: 5,
		ChanSize:  256,
	})
	err := w.Start(fn)
	if err != nil {
		t.Error(err)
	}
	for i := 1; i <= randnum; i++ {
		if err := w.Put(1); err != nil {
			t.Error(err)
		}
	}
	w.Close()
	w.Wait()
	if c != randnum {
		t.Fail()
	}
}

func TestWorker_Func_2(t *testing.T) {
	min := 15
	max := 4096
	randnum := rand.Intn(max-min) + min
	var c int
	mu := &sync.Mutex{}
	fn := func(single interface{}) {
		mu.Lock()
		defer mu.Unlock()
		if _, ok := single.(int); !ok {
			t.Error("Could not determine original value")
		}
		c += 1
	}
	w := New(&Options{
		Workers:   3,
		BatchSize: 5,
		ChanSize:  256,
	})
	err := w.Start(fn)
	if err != nil {
		t.Error(err)
	}
	for i := 1; i <= randnum; i++ {
		if err := w.Put(1); err != nil {
			t.Error(err)
		}
	}
	w.Close()
	w.Wait()
	if c != randnum {
		t.Fail()
	}
}

func TestWorker_FuncBatch_1(t *testing.T) {
	min := 15
	max := 4096
	randnum := rand.Intn(max-min) + min
	var c int
	mu := &sync.Mutex{}
	fn := FuncBatch(func(batch []interface{}) {
		for _, iface := range batch {
			if val, ok := iface.(int); ok {
				mu.Lock()
				c += val
				mu.Unlock()
			} else {
				t.Error("Could not determine original value")
			}
		}
	})
	w := New(&Options{
		Workers:   5,
		BatchSize: 3,
		ChanSize:  25,
	})
	err := w.Start(fn)
	if err != nil {
		t.Error(err)
	}
	for i := 1; i <= randnum; i++ {
		if err := w.Put(1); err != nil {
			t.Error(err)
		}
	}
	w.Close()
	if c != randnum {
		t.Fail()
	}
}

func TestWorker_FuncBatch_2(t *testing.T) {
	min := 15
	max := 4096
	randnum := rand.Intn(max-min) + min
	var c int
	mu := &sync.Mutex{}
	fn := func(batch []interface{}) {
		for _, iface := range batch {
			if val, ok := iface.(int); ok {
				mu.Lock()
				c += val
				mu.Unlock()
			} else {
				t.Error("Could not determine original value")
			}
		}
	}
	w := New(&Options{
		Workers:   3,
		BatchSize: 5,
		ChanSize:  256,
	})
	err := w.Start(fn)
	if err != nil {
		t.Error(err)
	}
	for i := 1; i <= randnum; i++ {
		if err := w.Put(1); err != nil {
			t.Error(err)
		}
	}
	w.Close()
	if c != randnum {
		t.Fail()
	}
}

func TestWorker_Close(t *testing.T) {
	min := 15
	max := 4096
	randnum := rand.Intn(max-min) + min
	var c int
	mu := &sync.Mutex{}
	fn := func(batch []interface{}) {
		for _, iface := range batch {
			if val, ok := iface.(int); ok {
				mu.Lock()
				c += val
				mu.Unlock()
			} else {
				t.Error("Could not determine original value")
			}
		}
	}
	batchsz := uint(randnum * 2)
	w := New(&Options{
		Workers:   1,
		BatchSize: batchsz,
		ChanSize:  256,
	})
	err := w.Start(fn)
	if err != nil {
		t.Error(err)
	}
	for i := 1; i <= randnum; i++ {
		if err := w.Put(1); err != nil {
			t.Error(err)
		}
	}
	if c > 0 {
		t.Errorf("Expected a zero workers before Flush() got %d", c)
	}
	w.Close()
	if c != randnum {
		t.Errorf("Expected generated number %d to match calculated number %d", randnum, c)
	}
}

func TestWorker_Flush(t *testing.T) {
	min := 15
	max := 4096
	randnum := rand.Intn(max-min) + min
	var c int
	mu := &sync.Mutex{}
	batchFunc := func(batch []interface{}) {
		mu.Lock()
		defer mu.Unlock()
		for _, iface := range batch {
			if val, ok := iface.(int); ok {
				c += val
			} else {
				t.Error("Could not determine original value")
			}
		}
	}
	batchsz := uint(randnum * 2)
	w := New(&Options{
		Workers:   1,
		BatchSize: batchsz,
		ChanSize:  256,
	})
	err := w.Start(batchFunc)
	if err != nil {
		t.Error(err)
	}
	for i := 1; i <= randnum; i++ {
		if err := w.Put(1); err != nil {
			t.Error(err)
		}
	}
	if c > 0 {
		t.Errorf("Expected a zero workers before Flush() got %d", c)
	}
	w.Flush()
	if c != randnum {
		t.Errorf("Expected generated number %d to match calculated number %d", randnum, c)
	}
	for i := 1; i <= randnum; i++ {
		if err := w.Put(1); err != nil {
			t.Error(err)
		}
	}
	w.Close()
	if c != randnum*2 {
		t.Errorf("Expected generated number %d to match calculated number %d", randnum, c)
	}
}

func TestWorker_Halt(t *testing.T) {
	c := 0
	fillsize := 100
	batchsz := 200
	fn := func(batch []interface{}) {
		for _, iface := range batch {
			if val, ok := iface.(int); ok {
				c += val
			} else {
				t.Error("Could not determine original value")
			}
		}
	}
	w := New(&Options{
		Workers:   1,
		BatchSize: uint(batchsz),
		ChanSize:  256,
	})
	w.Start(fn)
	for i := 0; i < fillsize; i++ {
		if err := w.Put(1); err != nil {
			t.Error(err)
		}
	}
	w.Flush()
	if c != fillsize {
		t.Errorf("fill size does not equal batch size: %d = %d", c, fillsize)
	}
	for i := 0; i < fillsize; i++ {
		if err := w.Put(1); err != nil {
			t.Error(err)
		}
	}
	w.Halt()
	if c != fillsize {
		t.Errorf("fill size does not equal batch size: %d = %d", c, fillsize)
	}
}
