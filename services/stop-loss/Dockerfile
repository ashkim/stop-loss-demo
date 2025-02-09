FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o stop-loss ./services/stop-loss

# Database initialization (using sqlite3)
RUN apk add --no-cache sqlite
COPY /services/stop-loss/migrations /migrations
RUN sqlite3 orders.db < /migrations/00-create-orders.sql

EXPOSE 8080

CMD ["./stop-loss"]
