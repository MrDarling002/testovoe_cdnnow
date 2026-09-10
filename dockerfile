FROM golang:1.22-bookworm AS builder

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable
ENV PATH="/root/.cargo/bin:${PATH}"

WORKDIR /src
COPY . .

RUN cd native/rust && cargo build --release

RUN CGO_ENABLED=1 go build -o /out/calculator ./cmd/calculator
RUN CGO_ENABLED=1 go build -o /out/generator ./cmd/generator

FROM debian:bookworm-slim
COPY --from=builder /out/calculator /usr/local/bin/calculator
COPY --from=builder /out/generator /usr/local/bin/generator
EXPOSE 8080
ENTRYPOINT ["calculator"]