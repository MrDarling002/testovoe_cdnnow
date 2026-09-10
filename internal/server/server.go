// Пакет server реализует HTTP-обработчики сервиса калькулятора.
package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"calculator/internal/metrics"
	"calculator/internal/native"
)

// Server хранит состояние калькулятора и рекордеры метрик.
type Server struct {
	sum int64 
	sub int64

	rps         *metrics.RPSRecorder
	latencyC    *metrics.LatencyRecorder
	latencyRust *metrics.LatencyRecorder
}

func New() *Server {
	return &Server{
		rps:         &metrics.RPSRecorder{},
		latencyC:    &metrics.LatencyRecorder{},
		latencyRust: &metrics.LatencyRecorder{},
	}
}

func (s *Server) Sum() int64 { return atomic.LoadInt64(&s.sum) }

func (s *Server) Sub() int64 { return atomic.LoadInt64(&s.sub) }

// Мы вызываем каждую нативную функцию ровно один раз с a = 0, чтобы
// выполнить обязательное CPU-вычисление и получить дельту. Дельта
// применяется через atomic.AddInt64. Это гарантирует:
//   - ровно один нативный вызов на запрос (нет CAS-retry),
//   - отсутствие сериализации между запросами,
//   - O(1) обновление состояния после нативного вызова.
func (s *Server) HandleCalc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	numStr := r.URL.Query().Get("num")
	if numStr == "" {
		http.Error(w, "missing num parameter", http.StatusBadRequest)
		return
	}
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid num parameter", http.StatusBadRequest)
		return
	}

	s.rps.Record()

	start := time.Now()
	delta := native.Add(0, num) // возвращает 0 + num = num
	s.latencyC.Record(time.Since(start))
	atomic.AddInt64(&s.sum, delta)

	start = time.Now()
	delta = native.Sub(0, num) // возвращает 0 - num = -num
	s.latencyRust.Record(time.Since(start))
	atomic.AddInt64(&s.sub, delta)

	w.WriteHeader(http.StatusOK)
}

// Значения latency являются best-effort snapshot последних 8192 сэмплов.
func (s *Server) HandleMetrics(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

    var buf strings.Builder
    buf.Grow(4096) // предварительное выделение памяти

    // RPS
    buf.WriteString("# HELP calculator_http_requests_per_second Запросов в секунду к /calc\n")
    buf.WriteString("# TYPE calculator_http_requests_per_second gauge\n")
    snap := s.rps.Snapshot()
    for i := 0; i < metrics.RPSWindowSize; i++ {
        fmt.Fprintf(&buf, "calculator_http_requests_per_second{second_offset=\"%d\"} %d\n", i, snap[i])
    }

    // Latency
    buf.WriteString("# HELP calculator_c_call_p95_seconds p95 длительность вызова C add()\n")
    buf.WriteString("# TYPE calculator_c_call_p95_seconds gauge\n")
    fmt.Fprintf(&buf, "calculator_c_call_p95_seconds %g\n", s.latencyC.Percentile(95).Seconds())

    buf.WriteString("# HELP calculator_c_call_p99_seconds p99 длительность вызова C add()\n")
    buf.WriteString("# TYPE calculator_c_call_p99_seconds gauge\n")
    fmt.Fprintf(&buf, "calculator_c_call_p99_seconds %g\n", s.latencyC.Percentile(99).Seconds())

    buf.WriteString("# HELP calculator_rust_call_p95_seconds p95 длительность вызова Rust sub()\n")
    buf.WriteString("# TYPE calculator_rust_call_p95_seconds gauge\n")
    fmt.Fprintf(&buf, "calculator_rust_call_p95_seconds %g\n", s.latencyRust.Percentile(95).Seconds())

    buf.WriteString("# HELP calculator_rust_call_p99_seconds p99 длительность вызова Rust sub()\n")
    buf.WriteString("# TYPE calculator_rust_call_p99_seconds gauge\n")
    fmt.Fprintf(&buf, "calculator_rust_call_p99_seconds %g\n", s.latencyRust.Percentile(99).Seconds())

    w.Write([]byte(buf.String()))
}