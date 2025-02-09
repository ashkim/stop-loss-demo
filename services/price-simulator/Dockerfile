FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o price-simulator ./services/price-simulator

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/price-simulator .
EXPOSE 8080
CMD ["./price-simulator"]
