FROM golang:1.22-alpine AS builder

RUN apk add --no-cache build-base clang clang-dev linux-headers

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p engine/build
RUN clang++ -shared -fPIC -O2 -std=c++17 -I engine/include engine/src/markov_chain.cpp engine/src/decision_engine.cpp engine/src/engine.cpp -o engine/build/libvektor_engine.so

ENV CGO_ENABLED=1
RUN go build -o proxy cmd/proxy/main.go
RUN go build -o coordinator cmd/coordinator/main.go

FROM alpine:latest
RUN apk add --no-cache libstdc++
WORKDIR /app
COPY --from=builder /app/engine/build/libvektor_engine.so /usr/local/lib/
ENV LD_LIBRARY_PATH=/usr/local/lib
RUN ldconfig /usr/local/lib || true

COPY --from=builder /app/proxy /app/
COPY --from=builder /app/coordinator /app/
COPY configs/ configs/
