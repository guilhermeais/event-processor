LOCALSTACK_CONTAINER := $(shell docker compose ps -q localstack)

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

send-msg:
	docker compose exec localstack awslocal sqs send-message \
		--queue-url http://localhost:4566/000000000000/event-queue \
		--message-body '{"id":"123432","birthday": "2026-08-26"}' \
		--message-attributes '{"client_id": {"DataType": "String", "StringValue": "client-123"},"event_id": {"DataType": "String", "StringValue": "'$$(uuidgen -r)'"}, "event_type": {"DataType": "String", "StringValue": "USER_CREATED"}}'