FROM golang:1.25.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker


FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/app .
COPY --from=builder /app/worker .

COPY config.yaml .
COPY internal/pb/swagger ./swagger

EXPOSE 8080
EXPOSE 50051

CMD ["./app"]
