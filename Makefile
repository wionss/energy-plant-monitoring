-include .env
export

.PHONY: dev dev-modd docker-up docker-down migrate-create goose-up goose-down goose-down-to goose-status hash install-dev-tools build run test swagger install-swagger webhook-receiver

# Development
dev:
	swag init -g main.go -o docs && air
run2:
	docker compose --profile dev up -d --build

dev-modd:
	swag init -g main.go -o docs && modd

run:
	swag init -g main.go -o docs && go run .

# Swagger
swagger:
	swag init -g main.go -o docs

install-swagger:
	go install github.com/swaggo/swag/cmd/swag@latest

build:
	go build -o ./tmp/main .

test:
	go test ./...

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Database Migrations
migrate-create:
	atlas migrate diff $(name) --env local

goose-up:
	goose -dir ./migrations postgres "$(DATABASE_URL)" up

goose-down:
	goose -dir ./migrations postgres "$(DATABASE_URL)" down

goose-down-to:
	goose -dir ./migrations postgres "$(DATABASE_URL)" down-to $(version)

goose-status:
	goose -dir ./migrations postgres "$(DATABASE_URL)" status

hash:
	atlas migrate hash --env local

console-kafka:
	docker exec -it monitoring-energy-kafka /bin/bash

# Webhook testing
webhook-receiver:
	go run ./cmd/webhook-receiver

webhock-run:
  PORT=9091 go run ./cmd/webhook-receiver
# Dev Tools Installation
install-dev-tools:
	go install github.com/air-verse/air@latest
	go install github.com/cortesi/modd/cmd/modd@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Installing Atlas..."
	@curl -sSf https://atlasgo.sh | sh

# Generate
generate:
	go generate ./...

zip:
	git archive --format=zip --output=service_$(shell git describe --tags --always --dirty).zip $(shell git write-tree)




