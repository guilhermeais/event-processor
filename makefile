PRODUCER_BIN = bin/produtor
PRODUCER_MAIN = ./producer.go
QUEUE_URL=http://localhost:4566/000000000000/event-queue

COVER_IGNORE='mocks/|tests/|cmd/'

test:
	go test -v ./...

coverage:
	go test -coverpkg=./... -coverprofile=coverage.tmp ./...
	grep -v -E $(COVER_IGNORE) coverage.tmp > coverage.out || true
	rm coverage.tmp
	go tool cover -func=coverage.out

coverage-html: coverage
	go tool cover -html=coverage.out

clean-test:
	rm -f coverage.out coverage.tmp

build:
	@echo "building..."
	mkdir -p build
	docker run --rm \
		-v "$(PWD):/app" \
		-w /app \
		-e GOOS=linux \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=0 \
		golang:1.25 \
		go build -tags lambda.norpc -o build/bootstrap cmd/lambda/main.go
	cd build && zip function.zip bootstrap

up: build
	chmod +x localstack/init-aws.sh
	docker compose up -d

down:
	-docker compose down -v
	rm -rf build/

logs:
	docker compose logs -f localstack

logs-cw:
	docker compose exec localstack awslocal logs filter-log-events \
		--log-group-name /aws/lambda/event-processor-lambda \
		--query 'events[*].message' \
		--output text

run-producer:
	AWS_REGION=us-east-1 \
	AWS_ACCESS_KEY_ID=test \
	AWS_SECRET_ACCESS_KEY=test \
	AWS_ENDPOINT_URL="http://localhost:4566" \
	QUEUE_URL=${QUEUE_URL} \
	go run $(PRODUCER_MAIN)