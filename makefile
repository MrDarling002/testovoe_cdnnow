.PHONY: build test race benchmark run-calculator run-generator clean

RUST_LIB := native/rust/target/release/libcalculator_rust.a

build: $(RUST_LIB)
	CGO_ENABLED=1 go build -o bin/calculator ./cmd/calculator
	CGO_ENABLED=1 go build -o bin/generator ./cmd/generator

$(RUST_LIB): native/rust/src/lib.rs native/rust/Cargo.toml
	cd native/rust && cargo build --release

test: $(RUST_LIB)
	go test -v -count=1 ./...

race: $(RUST_LIB)
	go test -race -count=1 ./...

benchmark: $(RUST_LIB)
	go test -bench=. -benchmem -count=1 ./internal/server/

run-calculator: build
	./bin/calculator

run-generator: build
	./bin/generator -workers 100 -url http://localhost:8080

clean:
	rm -rf bin/
	cd native/rust && cargo clean