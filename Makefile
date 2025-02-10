.PHONY: all wscat-prices run test clean lint init sql dump-orders up

DB_FILE := ./services/stop-loss/data/orders.db  

all: up


up: init
	docker-compose up --build

test:
	go test -v ./...

clean:
	docker-compose down -v

	# delete this guy or we'll be out of sync with temporal
	rm -f $(DB_FILE)  

lint:
	golangci-lint run

init:
	go mod tidy
	go mod verify

sql:
	@sqlite3 $(DB_FILE)

dump-orders:
	@sqlite3 $(DB_FILE) "SELECT * FROM orders;" 

wscat-prices:
	wscat -c ws://localhost:8081/prices
