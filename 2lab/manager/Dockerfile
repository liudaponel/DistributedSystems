FROM golang:1.23.0 AS builder
WORKDIR /app

COPY . ./manager/

WORKDIR /app/manager

RUN go mod download

RUN go build -o manager .

FROM debian:latest
WORKDIR /app

COPY --from=builder /app/manager/manager .

EXPOSE 8080
CMD ["./manager"]
