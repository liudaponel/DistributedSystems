FROM golang:1.23.0 AS builder

WORKDIR /app
COPY . ./worker/

WORKDIR /app/worker

RUN go mod download

RUN go build -o worker .

FROM debian:latest
WORKDIR /app

COPY --from=builder /app/worker/worker .

EXPOSE 8081
CMD ["./worker"]
