package worker

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestWorker_Start_1(t *testing.T) {
	fillcnt := 24
	var count int
	w := New(nil)
	fn := func(single interface{}) {
		i, ok := single.(int)
		if !ok {
			t.Error("Could not determine original value")
			return
		}
		count += i
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
		t.Errorf("Counts do not match %d = %d", fillcnt, count)
	}
}

func TestWorker_Start_2(t *testing.T) {
	w := New(nil)
	err := w.Start(func() {})
	if err == nil {
		t.Error("Expecting unknown type")
	}

}

func Test_start(t *testing.T) {
	w := New(nil)
	ch := make(chan interface{}, 1)
	defer func() {
		if r := recover(); r == nil {
			t.Fail()
		}
	}()
	start(w, ch, func() {})
}

func TestWorker_Put_1(t *testing.T) {
	var count int
	fn := func(single interface{}) {
		i, ok := single.(int)
		if !ok {
			t.Error("Expecting an int got")
			return
		}
		count += i
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
		i, ok := single.(int)
		if !ok {
			t.Error("Expecting an int got")
			return
		}
		count += i
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
	data := []interface{}{20, 20, 20}
	w := New(nil)
	if err := w.Start(fn); err != nil {
		t.Errorf("Got an error: %v", err)
	}
	for _, payload := range data {
		if err := w.Put(payload); err != nil {
			t.Errorf("Got an error: %v", err)
		}
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
		c++
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
		c++
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
		t.Fatalf("Result %d does not match random number %d", c, randnum)
	}
}
func TestWorker_FuncContext_1(t *testing.T) {
	fn := FuncContext(func(ctx context.Context, v interface{}) {
		tout, ok := v.(int)
		if !ok {
			panic("expected an integer")
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(tout) * time.Second):
				t.FailNow()
				return
			}
		}

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
	ctx, cancel := context.WithCancel(context.Background())
	w.PutC(ctx, 30)
	w.chanWG.Wait()
	cancel()
	w.Close()
}

func TestWorker_FuncContext_2(t *testing.T) {
	fn := func(ctx context.Context, v interface{}) {
		tout, ok := v.(int)
		if !ok {
			panic("expected an integer")
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(tout) * time.Second):
				t.FailNow()
				return
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
	ctx, cancel := context.WithCancel(context.Background())
	w.PutC(ctx, 30)
	w.chanWG.Wait()
	cancel()
	w.Close()
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

func TestWorker_FuncBatchContext_1(t *testing.T) {
	fn := FuncBatchContext(func(ctx context.Context, batch []interface{}) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				t.FailNow()
				return
			}
		}
	})
	w := New(&Options{
		Workers:   3,
		BatchSize: 1,
		ChanSize:  256,
	})
	err := w.Start(fn)
	if err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	for i := 1; i <= 3; i++ {
		if err := w.PutC(ctx, 1); err != nil {
			t.Error(err)
		}
	}
	cancel()
	w.Close()
}

func TestWorker_FuncBatchContext_2(t *testing.T) {
	fn := func(ctx context.Context, batch []interface{}) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				t.FailNow()
				return
			}
		}
	}
	w := New(&Options{
		Workers:   3,
		BatchSize: 1,
		ChanSize:  256,
	})
	err := w.Start(fn)
	if err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := w.PutC(ctx, 1); err != nil {
		t.Error(err)
	}
	cancel()

	w.Close()
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

func TestWorker_Flush_1(t *testing.T) {
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

func TestWorker_Flush_2(t *testing.T) {
	min := 15
	max := 4096
	randnum := rand.Intn(max-min) + min
	var c int
	mu := &sync.Mutex{}
	batchFunc := func(single interface{}) {
		mu.Lock()
		defer mu.Unlock()
		i, ok := single.(int)
		if !ok {
			t.Error("Expecting an int got")
			return
		}
		c += i
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
