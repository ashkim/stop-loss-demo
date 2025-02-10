.PHONY: all wscat-prices run test clean lint init

all: run

wscat-prices:
	wscat -c ws://localhost:8081/prices

up: init
	docker-compose up --build

test:
	go test -v ./...

clean:
	docker-compose down -v

	# delete this guy or we'll be out of sync with temporal 
	rm -f ./services/stop-loss/data/orders.db

lint:
	golangci-lint run

init:
	go mod tidy
	go mod verify
