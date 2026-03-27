.PHONY: test coverage coverage-html clean-test build up down logs logs-cw tf-init tf-apply tf-destroy run-producer seed

PRODUCER_BIN = bin/produtor
PRODUCER_MAIN = ./producer.go
COVER_IGNORE='mocks/|tests/|cmd/'

TF_DOCKER = docker run --rm -v "$(PWD):/app" -w /app/terraform --network host hashicorp/terraform:latest
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
	$(eval QUEUE_URL := $(shell $(TF_DOCKER) output -raw queue_url))
	@echo "URL da fila: $(QUEUE_URL)"

	AWS_REGION=us-east-1 \
	AWS_ACCESS_KEY_ID=test \
	AWS_SECRET_ACCESS_KEY=test \
	AWS_ENDPOINT_URL="http://localhost:4566" \
	QUEUE_URL=${QUEUE_URL} \
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