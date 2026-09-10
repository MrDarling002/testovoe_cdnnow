package metrics

import (
	"sort"
	"sync/atomic"
	"time"
)

const LatencyBufSize = 8192

// LatencyRecorder хранит последние LatencyBufSize значений длительности
// в кольцевом буфере без блокировок на пути записи.
// Память: 8192 × 8 байт = 64 КиБ на один экземпляр.
//
// Путь записи атомарный инкремент индекса + атомарная запись.
// Без аллокаций, без блокировок.
//
// Путь чтения (только /metrics): копирование последних min(total, BufSize)
// сэмплов, сортировка, выборка перцентиля. Это O(n log n) при n ≤ 8192,
type LatencyRecorder struct {
	buf [LatencyBufSize]int64 // длительности в наносекундах
	idx uint64               // общее число записей, монотонно растёт
}

func (l *LatencyRecorder) Record(d time.Duration) {
	i := atomic.AddUint64(&l.idx, 1) - 1
	atomic.StoreInt64(&l.buf[i%LatencyBufSize], int64(d))
}

func (l *LatencyRecorder) Percentile(p float64) time.Duration {
	end := atomic.LoadUint64(&l.idx)
	if end == 0 {
		return 0
	}

	n := int(end)
	if n > LatencyBufSize {
		n = LatencyBufSize
	}

	start := end - uint64(n)
	samples := make([]int64, n)
	for i := 0; i < n; i++ {
		samples[i] = atomic.LoadInt64(&l.buf[(start+uint64(i))%LatencyBufSize])
	}

	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })

	rank := int(float64(n-1) * p / 100.0)
	if rank >= n {
		rank = n - 1
	}
	return time.Duration(samples[rank])
}