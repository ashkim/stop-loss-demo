FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN apk add --no-cache build-base gcc musl-dev

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o stop-loss ./services/stop-loss

# Final image stage
FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache sqlite-libs  

COPY --from=builder /app/stop-loss .

EXPOSE 8080

CMD ["./stop-loss"]
