package server

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestCalcSingleRequest(t *testing.T) {
	s := New()

	req := httptest.NewRequest(http.MethodPost, "/calc?num=7", nil)
	w := httptest.NewRecorder()
	s.HandleCalc(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ожидалось 200, получено %d", w.Code)
	}
	if s.Sum() != 7 {
		t.Errorf("sum: ожидалось 7, получено %d", s.Sum())
	}
	if s.Sub() != -7 {
		t.Errorf("sub: ожидалось -7, получено %d", s.Sub())
	}
}

func TestCalcSequential(t *testing.T) {
	s := New()

	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/calc?num=%d", i), nil)
		w := httptest.NewRecorder()
		s.HandleCalc(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("запрос %d: ожидалось 200, получено %d", i, w.Code)
		}
	}

	if s.Sum() != 15 {
		t.Errorf("sum: ожидалось 15, получено %d", s.Sum())
	}
	if s.Sub() != -15 {
		t.Errorf("sub: ожидалось -15, получено %d", s.Sub())
	}
}

func TestCalcBoundaryValues(t *testing.T) {
	s := New()

	nums := []int64{-100, -1, 0, 1, 100}
	var expectedSum, expectedSub int64

	for _, num := range nums {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/calc?num=%d", num), nil)
		w := httptest.NewRecorder()
		s.HandleCalc(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("num=%d: ожидалось 200, получено %d", num, w.Code)
		}
		expectedSum += num
		expectedSub -= num
	}

	if s.Sum() != expectedSum {
		t.Errorf("sum: ожидалось %d, получено %d", expectedSum, s.Sum())
	}
	if s.Sub() != expectedSub {
		t.Errorf("sub: ожидалось %d, получено %d", expectedSub, s.Sub())
	}
}

func TestCalcConcurrentSameNum(t *testing.T) {
	s := New()
	const n = 200

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/calc?num=3", nil)
			w := httptest.NewRecorder()
			s.HandleCalc(w, req)
		}()
	}
	wg.Wait()

	if s.Sum() != int64(n*3) {
		t.Errorf("sum: ожидалось %d, получено %d", n*3, s.Sum())
	}
	if s.Sub() != -int64(n*3) {
		t.Errorf("sub: ожидалось %d, получено %d", -n*3, s.Sub())
	}
}

func TestCalcConcurrentMixedNumbers(t *testing.T) {
	s := New()
	const n = 500

	nums := make([]int64, n)
	var expectedSum, expectedSub int64
	rng := rand.New(rand.NewPCG(42, 7))
	for i := range nums {
		nums[i] = rng.Int64N(201) - 100
		expectedSum += nums[i]
		expectedSub -= nums[i]
	}

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(num int64) {
			defer wg.Done()
			url := fmt.Sprintf("/calc?num=%d", num)
			req := httptest.NewRequest(http.MethodPost, url, nil)
			w := httptest.NewRecorder()
			s.HandleCalc(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("num=%d: ожидалось 200, получено %d", num, w.Code)
			}
		}(nums[i])
	}
	wg.Wait()

	if s.Sum() != expectedSum {
		t.Errorf("sum: ожидалось %d, получено %d", expectedSum, s.Sum())
	}
	if s.Sub() != expectedSub {
		t.Errorf("sub: ожидалось %d, получено %d", expectedSub, s.Sub())
	}
}

func TestCalcMissingNum(t *testing.T) {
	s := New()
	req := httptest.NewRequest(http.MethodPost, "/calc", nil)
	w := httptest.NewRecorder()
	s.HandleCalc(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("ожидалось 400, получено %d", w.Code)
	}
}

func TestCalcInvalidNum(t *testing.T) {
	s := New()
	for _, bad := range []string{"abc", "12.5", "99999999999999999999999"} {
		req := httptest.NewRequest(http.MethodPost, "/calc?num="+bad, nil)
		w := httptest.NewRecorder()
		s.HandleCalc(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("num=%q: ожидалось 400, получено %d", bad, w.Code)
		}
	}
}

func TestCalcWrongMethod(t *testing.T) {
	s := New()
	req := httptest.NewRequest(http.MethodGet, "/calc?num=5", nil)
	w := httptest.NewRecorder()
	s.HandleCalc(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("ожидалось 405, получено %d", w.Code)
	}
}

func TestMetricsFormat(t *testing.T) {
	s := New()

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/calc?num=2", nil)
		w := httptest.NewRecorder()
		s.HandleCalc(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	s.HandleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ожидалось 200, получено %d", w.Code)
	}

	body := w.Body.String()
	for _, expected := range []string{
		"calculator_http_requests_per_second",
		"calculator_c_call_p95_seconds",
		"calculator_c_call_p99_seconds",
		"calculator_rust_call_p95_seconds",
		"calculator_rust_call_p99_seconds",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("в выводе метрик отсутствует %q", expected)
		}
	}
}

func TestMetricsConcurrentWithWrites(t *testing.T) {
	s := New()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest(http.MethodPost, "/calc?num=1", nil)
			w := httptest.NewRecorder()
			s.HandleCalc(w, req)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			w := httptest.NewRecorder()
			s.HandleMetrics(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("metrics вернул %d", w.Code)
			}
		}
	}()

	wg.Wait()
}

func BenchmarkCalcHandler(b *testing.B) {
	s := New()
	req := httptest.NewRequest(http.MethodPost, "/calc?num=42", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.HandleCalc(w, req)
	}
}

func BenchmarkCalcConcurrent(b *testing.B) {
	s := New()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodPost, "/calc?num=1", nil)
			w := httptest.NewRecorder()
			s.HandleCalc(w, req)
		}
	})
}