
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Add this RUN command to list the contents of the html directory in the builder
RUN echo "Listing contents of /app/services/stop-loss/html in builder:"
RUN ls -lR /app/services/stop-loss/html

RUN CGO_ENABLED=0 GOOS=linux go build -o stop-loss ./services/stop-loss

# Final image stage
FROM alpine:3.18

WORKDIR /app

# Database initialization (using sqlite3)
RUN apk add --no-cache sqlite

COPY --from=builder /app/stop-loss .
COPY --from=builder /app/services/stop-loss/migrations /app/migrations
COPY --from=builder /app/services/stop-loss/entrypoint.sh /app/entrypoint.sh

RUN mkdir -p /app/data
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]

EXPOSE 8080

CMD ["./stop-loss"]
