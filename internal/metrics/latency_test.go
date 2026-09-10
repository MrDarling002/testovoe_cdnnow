package metrics

import (
	"testing"
	"time"
)

func TestLatencyPercentileBasic(t *testing.T) {
	l := &LatencyRecorder{}

	for i := 1; i <= 100; i++ {
		l.Record(time.Duration(i) * time.Millisecond)
	}

	p95 := l.Percentile(95)
	p99 := l.Percentile(99)

	if p95 < 90*time.Millisecond || p95 > 100*time.Millisecond {
		t.Errorf("p95 = %v, ожидалось ~95ms", p95)
	}
	if p99 < 95*time.Millisecond || p99 > 100*time.Millisecond {
		t.Errorf("p99 = %v, ожидалось ~99ms", p99)
	}
}

func TestLatencyEmpty(t *testing.T) {
	l := &LatencyRecorder{}
	if l.Percentile(95) != 0 {
		t.Error("ожидалось 0 для пустого рекордера")
	}
}

func TestLatencySingleSample(t *testing.T) {
	l := &LatencyRecorder{}
	l.Record(42 * time.Millisecond)

	if l.Percentile(95) != 42*time.Millisecond {
		t.Errorf("один сэмпл: ожидалось 42ms, получено %v", l.Percentile(95))
	}
}

func TestLatencyRingWrapAround(t *testing.T) {
	l := &LatencyRecorder{}

	total := LatencyBufSize + 1000
	for i := 0; i < total; i++ {
		l.Record(time.Duration(i+1) * time.Nanosecond)
	}

	// Последние LatencyBufSize сэмплов: [1001 .. 9192] нс.
	// Медиана должна быть в середине этого диапазона.
	p50 := l.Percentile(50)
	expectedMedian := time.Duration(1000+LatencyBufSize/2) * time.Nanosecond
	tolerance := time.Duration(LatencyBufSize/4) * time.Nanosecond

	if p50 < expectedMedian-tolerance || p50 > expectedMedian+tolerance {
		t.Errorf("p50 = %v, ожидалось ~%v (ошибка обхода кольца?)", p50, expectedMedian)
	}
}

func TestLatencyConcurrentWrites(t *testing.T) {
	l := &LatencyRecorder{}
	const goroutines = 50
	const perG = 200

	done := make(chan struct{}, goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			for i := 0; i < perG; i++ {
				l.Record(time.Duration(i+1) * time.Microsecond)
			}
			done <- struct{}{}
		}()
	}
	for g := 0; g < goroutines; g++ {
		<-done
	}

	p99 := l.Percentile(99)
	if p99 <= 0 {
		t.Error("ожидалось положительное значение p99 после конкурентной записи")
	}
	if p99 > 200*time.Microsecond {
		t.Errorf("p99 = %v превышает максимально возможное 200мкс", p99)
	}
}