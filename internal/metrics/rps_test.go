package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestRPSRecordAndSnapshot(t *testing.T) {
	r := &RPSRecorder{}

	for i := 0; i < 5; i++ {
		r.Record()
	}

	snap := r.Snapshot()
	if snap[0] != 5 {
		t.Errorf("текущая секунда: ожидалось 5, получено %d", snap[0])
	}
}

func TestRPSConcurrent(t *testing.T) {
	r := &RPSRecorder{}
	const goroutines = 200
	const perG = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				r.Record()
			}
		}()
	}
	wg.Wait()

	snap := r.Snapshot()
	var total int64
	for _, v := range snap {
		total += v
	}
	want := int64(goroutines * perG)
	if total != want {
		t.Errorf("ожидалось %d запросов, получено %d", want, total)
	}
}

func TestRPSEmptyReturnsZeros(t *testing.T) {
	r := &RPSRecorder{}
	snap := r.Snapshot()
	for i, v := range snap {
		if v != 0 {
			t.Errorf("слот %d: ожидалось 0, получено %d", i, v)
		}
	}
}

func TestRPSStaleSlotIgnored(t *testing.T) {
	r := &RPSRecorder{}
	pastSec := time.Now().Unix() - 61
	idx := int(pastSec % RPSWindowSize)
	r.slots[idx].mu.Lock()
	r.slots[idx].sec = pastSec
	r.slots[idx].count = 999
	r.slots[idx].mu.Unlock()

	r.Record()

	snap := r.Snapshot()
	if snap[0] != 1 {
		t.Errorf("текущая секунда: ожидалось 1, получено %d", snap[0])
	}
	for i, v := range snap {
		if i == 0 {
			continue
		}
		if v == 999 {
			t.Errorf("устаревший слот попал в позицию %d", i)
		}
	}
}

func TestRPSConcurrentRollover(t *testing.T) {
	r := &RPSRecorder{}
	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			r.Record()
		}()
	}
	wg.Wait()

	snap := r.Snapshot()
	var total int64
	for _, v := range snap {
		total += v
	}
	if total != goroutines {
		t.Errorf("ожидалось %d, получено %d (потеря при переходе секунды?)", goroutines, total)
	}
}