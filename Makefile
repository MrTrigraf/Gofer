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
	migrate -path ./migration -database "postgres://gofer:gofer@localhost:5432/gofer?sslmode=disable" up

migrate-down:
	migrate -path ./migration -database "postgres://gofer:gofer@localhost:5432/gofer?sslmode=disable" down