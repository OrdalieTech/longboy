FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN apk add --no-cache gcc build-base
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/server .
COPY web ./web

EXPOSE 8080

CMD ["./server"]