#!/bin/bash
echo "Initializing LocalStack infra..."

awslocal dynamodb create-table \
    --table-name schemas \
    --attribute-definitions AttributeName=event_type,AttributeType=S \
    --key-schema AttributeName=event_type,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST

echo "creating default schema 'USER_CREATED'"
awslocal dynamodb put-item \
    --table-name schemas \
    --item '{
        "event_type": {"S": "USER_CREATED"},
        "schema": {"S": "{\"type\":\"object\",\"properties\": {\"id\": { \"type\": \"string\" },\"birthday\": { \"type\": \"string\", \"format\": \"date\" }}}"}
    }'

awslocal dynamodb create-table \
    --table-name events \
    --attribute-definitions \
        AttributeName=client_id,AttributeType=S \
        AttributeName=event_id,AttributeType=S \
    --key-schema \
        AttributeName=client_id,KeyType=HASH \
        AttributeName=event_id,KeyType=RANGE \
    --billing-mode PAY_PER_REQUEST

awslocal sqs create-queue --queue-name event-dlq

awslocal sqs create-queue \
    --queue-name event-queue \
    --attributes '{
      "RedrivePolicy": "{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:000000000000:event-dlq\",\"maxReceiveCount\":\"3\"}",
      "VisibilityTimeout": "30"
    }'

awslocal lambda create-function \
    --function-name event-processor-lambda \
    --runtime provided.al2023 \
    --handler bootstrap \
    --role arn:aws:iam::000000000000:role/dummy-role \
    --zip-file fileb:///var/task/function.zip \
    --environment Variables="{SCHEMA_TABLE_NAME=schemas,EVENTS_TABLE_NAME=events,DLQ_URL=http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/event-dlq}"

awslocal lambda create-event-source-mapping \
    --function-name event-processor-lambda \
    --batch-size 10 \
    --event-source-arn arn:aws:sqs:us-east-1:000000000000:event-queue \
    --function-response-types "ReportBatchItemFailures"

echo "LocalStack infraestructure initialized!"