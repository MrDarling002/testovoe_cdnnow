package native

/*
#cgo CFLAGS: -O2
#cgo LDFLAGS: -L${SRCDIR}/../../native/rust/target/release -lcalculator_rust -lm -ldl -lpthread
#include "calculator.h"
#include "rust_sub.h"
*/
import "C"

func Add(a, b int64) int64 {
	return int64(C.add(C.int64_t(a), C.int64_t(b)))
}

func Sub(a, b int64) int64 {
	return int64(C.sub(C.int64_t(a), C.int64_t(b)))
}