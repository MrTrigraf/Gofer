.PHONY: docker docker-down run-server run-client migrate-up migrate-down

docker:
	docker-compose up -d

docker-down:
	docker-compose down

run-server:
	go run ./cmd/server/...

run-client:
	go run ./cmd/client/...

migrate-up:
	go run ./cmd/migrate/up

migrate-down:
	go run ./cmd/migrate/down