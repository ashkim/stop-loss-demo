.PHONY: build run test clean price-sim stop-loss

# Default target
all: build

# Build all services
build: price-sim stop-loss

# Build price simulator
price-sim:
	go build -o bin/price-simulator ./services/price-simulator

# Build stop loss service (placeholder)
 stop-loss:
# 	go build -o bin/stop-loss ./services/stop-loss

# Run price simulator directly
run-price-sim: price-sim
	./bin/price-simulator

wscat-prices:
	wscat -c ws://localhost:8081/prices

# Run stop loss service directly
run-stop-loss: stop-loss
	#./bin/stop-loss

# Run all services via docker-compose
run:
	docker-compose up --build

# Run in detached mode
run-d:
	docker-compose up -d --build

# Stop services
stop:
	docker-compose down

# Run tests
test:
	go test -v ./...

# Clean built binaries
clean:
	rm -rf bin/
	docker-compose down -v

# Linting
lint:
	golangci-lint run

# Initialize development environment
init:
	mkdir -p bin
	go mod tidy
	go mod verify

refresh-stop-loss:
	docker-compose down stop-loss  # Stop and remove the stop-loss container
	docker-compose build stop-loss # Rebuild the stop-loss image
	docker-compose up -d stop-loss   # Start the stop-loss container
