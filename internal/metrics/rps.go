package metrics

import (
	"sync"
	"time"
)

const RPSWindowSize = 60

type rpsSlot struct {
	mu    sync.Mutex
	sec   int64
	count int64
}

// RPSRecorder отслеживает количество запросов по секундам в фиксированном
type RPSRecorder struct {
	slots [RPSWindowSize]rpsSlot
}

func (r *RPSRecorder) Record() {
	sec := time.Now().Unix()
	idx := int(sec % RPSWindowSize)
	s := &r.slots[idx]

	s.mu.Lock()
	if s.sec != sec {
		s.sec = sec
		s.count = 0
	}
	s.count++
	s.mu.Unlock()
}

// Snapshot возвращает количество запросов за последние 60 секунд.
func (r *RPSRecorder) Snapshot() [RPSWindowSize]int64 {
	now := time.Now().Unix()
	var result [RPSWindowSize]int64

	for i := 0; i < RPSWindowSize; i++ {
		s := &r.slots[i]
		s.mu.Lock()
		sec := s.sec
		count := s.count
		s.mu.Unlock()

		if sec == 0 {
			continue
		}
		age := now - sec
		if age >= 0 && age < RPSWindowSize {
			result[age] = count
		}
	}
	return result
}

// Reset очищает все слоты (для тестов).
func (r *RPSRecorder) Reset() {
	for i := range r.slots {
		r.slots[i].mu.Lock()
		r.slots[i].sec = 0
		r.slots[i].count = 0
		r.slots[i].mu.Unlock()
	}
}