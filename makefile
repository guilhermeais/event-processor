.PHONY: test coverage coverage-html clean-test build up down logs logs-cw tf-init tf-apply tf-destroy run-producer seed diag diag-sns diag-sqs diag-dlq diag-dynamo clean-queues clean-sqs clean-dlq

PRODUCER_MAIN = ./cmd/producer/main.go
COVER_IGNORE='mocks/|tests/|cmd/'

TF_DOCKER = docker run --rm -v "$(PWD):/app" -w /app/terraform/envs/local --network host hashicorp/terraform:latest
PY_DOCKER = docker run --rm -it -v "$(PWD):/app" -w /app --network host

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
	@echo "Subindo a infraestrutura e aguardando LocalStack ficar pronto..."
	docker compose up -d --wait
	@echo "LocalStack está saudável! Iniciando o Terraform..."
	$(MAKE) tf-apply
	$(MAKE) seed

down:
	$(MAKE) tf-destroy
	-docker compose down -v
	rm -rf build/

logs:
	docker compose logs -f localstack

logs-cw:
	@echo "Coletando outputs do Terraform..."
	$(eval LAMBDA_NAME := $(shell $(TF_DOCKER) output -raw lambda_name))
	@echo "Nome da função Lambda: $(LAMBDA_NAME)"
	docker compose exec localstack awslocal logs filter-log-events \
		--log-group-name /aws/lambda/$(LAMBDA_NAME) \
		--query 'events[*].message' \
		--output text

diag: diag-sns diag-sqs diag-dlq diag-dynamo

diag-sns:
	@echo "=== SNS: subscriptions ==="
	$(eval TOPIC_ARN := $(shell $(TF_DOCKER) output -raw event_topic_arn))
	docker compose exec localstack awslocal sns list-subscriptions-by-topic \
		--topic-arn $(TOPIC_ARN) \
		--query 'Subscriptions[*].{Protocol:Protocol,Endpoint:Endpoint,SubscriptionArn:SubscriptionArn}' \
		--output table

diag-sqs:
	@echo "=== SQS: mensagens na fila ==="
	$(eval QUEUE_URL := $(shell $(TF_DOCKER) output -raw queue_url))
	docker compose exec localstack awslocal sqs get-queue-attributes \
		--queue-url $(QUEUE_URL) \
		--attribute-names ApproximateNumberOfMessages ApproximateNumberOfMessagesNotVisible ApproximateNumberOfMessagesDelayed \
		--query 'Attributes' \
		--output table

diag-dlq:
	@echo "=== DLQ: contagem e preview ==="
	$(eval DLQ_URL := $(shell $(TF_DOCKER) output -raw dlq_url))
	docker compose exec localstack awslocal sqs get-queue-attributes \
		--queue-url $(DLQ_URL) \
		--attribute-names ApproximateNumberOfMessages \
		--query 'Attributes' \
		--output table
	@echo "--- mensagens na DLQ (peek, visibility=30s) ---"
	docker compose exec localstack awslocal sqs receive-message \
		--queue-url $(DLQ_URL) \
		--max-number-of-messages 5 \
		--message-attribute-names All \
		--visibility-timeout 30 \
		--query 'Messages[*].{Id:MessageId,Attributes:MessageAttributes,Body:Body}' \
		--output json

clean-queues: clean-sqs clean-dlq
	@echo "Filas limpas!"

clean-sqs:
	@echo "Limpando SQS event-queue..."
	$(eval QUEUE_URL := $(shell $(TF_DOCKER) output -raw queue_url))
	docker compose exec localstack awslocal sqs purge-queue --queue-url $(QUEUE_URL)
	@echo "✓ SQS event-queue purgada"

clean-dlq:
	@echo "Limpando SQS event-dlq..."
	$(eval DLQ_URL := $(shell $(TF_DOCKER) output -raw dlq_url))
	docker compose exec localstack awslocal sqs purge-queue --queue-url $(DLQ_URL)
	@echo "✓ SQS event-dlq purgada"

tf-init:
	@echo "Inicializando Terraform via Docker..."
	$(TF_DOCKER) init

tf-apply: tf-init
	@echo "Aplicando infraestrutura..."
	$(TF_DOCKER) apply -auto-approve

tf-destroy:
	@echo "Destruindo infraestrutura..."
	$(TF_DOCKER) destroy -auto-approve

run-producer:
	@echo "Coletando outputs do Terraform..."
	$(eval TOPIC_ARN := $(shell $(TF_DOCKER) output -raw event_topic_arn))
	@echo "ARN do tópico: $(TOPIC_ARN)"

	AWS_REGION=us-east-1 \
	AWS_ACCESS_KEY_ID=test \
	AWS_SECRET_ACCESS_KEY=test \
	AWS_ENDPOINT_URL="http://localhost:4566" \
	TOPIC_ARN=${TOPIC_ARN} \
	go run $(PRODUCER_MAIN)

seed:
	@echo "Coletando outputs do Terraform..."
	$(eval SCHEMAS_TABLE := $(shell $(TF_DOCKER) output -raw schemas_table_name))
	
	@echo "Tabela de schemas encontrada: $(SCHEMAS_TABLE)"
	@echo "Inserindo schemas iniciais via Docker..."

	$(PY_DOCKER) \
		-e SCHEMAS_TABLE_NAME=$(SCHEMAS_TABLE) \
		-e AWS_ENDPOINT_URL=http://localhost:4566 \
		-e AWS_ACCESS_KEY_ID=test \
		-e AWS_SECRET_ACCESS_KEY=test \
		-e AWS_DEFAULT_REGION=us-east-1 \
		python:3.11-slim \
		bash -c "pip install boto3 -q && python scripts/seed.py"