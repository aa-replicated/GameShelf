.PHONY: build run dev clean test

# Build the Go binary
build:
	go build -o bin/gameshelf ./cmd/gameshelf

# Start all services with docker-compose
run:
	docker-compose up --build

# Run the dev server locally (requires DB + Redis already running)
dev:
	DATABASE_URL="postgres://gameshelf:gameshelf@localhost:5432/gameshelf?sslmode=disable" \
	REDIS_URL="redis://localhost:6379/0" \
	ADMIN_SECRET="devpassword" \
	SITE_NAME="GameShelf Dev" \
	go run ./cmd/gameshelf

# Run tests
test:
	go test ./...

# Remove built artifacts
clean:
	rm -rf bin/
	docker-compose down -v
