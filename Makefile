.PHONY: run build test lint tidy up down

run:
	go run ./cmd/redisforge

build:
	go build -o bin/redisforge ./cmd/redisforge

test:
	go test -race ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

up:
	docker compose -f deployments/docker-compose.yml up -d

down:
	docker compose -f deployments/docker-compose.yml down