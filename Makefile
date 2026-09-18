.PHONY: run test generate migrate-up docker-up docker-down build

run:
	go run .

test:
	go test ./...

generate:
	sqlc generate

migrate-up:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f migrations/0001_initial.sql

docker-up:
	docker compose up --build

docker-down:
	docker compose down

build:
	go build -o bin/gotcha .
