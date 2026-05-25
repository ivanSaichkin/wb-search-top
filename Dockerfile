FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o search_service ./cmd/app/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/search_service .

EXPOSE 8080
EXPOSE 2112

CMD ["./search_service"]