# RFC 0001: Lambda and Kubernetes SQS Consumers

- Status: Proposed
- Authors: Event Processor team
- Date: 2026-10-24

## Summary

Add a Kubernetes worker entrypoint to the event processor while preserving the existing AWS Lambda entrypoint. Both runtimes will consume the same SQS queue concurrently during the migration period and will share the complete processing core: schema loading and caching, validation, persistence, and the `ACK`, `RETRY`, and `TO_DLQ` decisions.

The initial deployment keeps Lambda enabled. Kubernetes becomes an additional consumer and can later become the only consumer after Lambda is deliberately disabled.

## Motivation

The current service runs as a Go Lambda connected to SQS through an Event Source Mapping. This provides managed scaling, but it does not provide a long-running worker deployment model. The project needs an operationally equivalent Kubernetes process without creating a second implementation of the event-processing rules.

## Goals

- Add a separate `cmd/worker` executable for Kubernetes.
- Preserve `cmd/lambda` and its SQS batch failure contract.
- Share schema loading, local schema caching, validation, persistence, and business decisions.
- Support concurrent Lambda and Kubernetes consumption from the same SQS queue.
- Preserve at-least-once delivery and idempotent persistence.
- Define polling, bounded concurrency, visibility handling, DLQ behavior, and graceful shutdown.
- Define the container, Kubernetes deployment, AWS permissions, configuration, observability, and rollout path.

## Non-goals

- Replacing SQS with Kafka, SNS, or another broker.
- Processing every message twice by giving Lambda and Kubernetes separate copies.
- Introducing a new validation or persistence model.
- Adding schema mutation APIs or cross-process cache invalidation.
- Removing Lambda in the first implementation.

## Current State

The Lambda runtime is built from `cmd/lambda/main.go` and packaged as `build/bootstrap`. Terraform connects the function to SQS with batch processing and `ReportBatchItemFailures`. The Lambda adapter extracts `client_id`, `event_id`, and `event_type`, invokes the processor, and reports retryable batch item failures or sends terminal failures to the DLQ.

The processing core loads schemas from DynamoDB, validates JSON payloads, and persists accepted events to DynamoDB using `client_id#event_id` as the idempotency key. There is currently no Kubernetes executable, application container, worker polling loop, or Kubernetes deployment.

## Proposed Architecture

```text
                         +------------------+
                         |    SQS queue     |
                         +--------+---------+
                                  |
                 +----------------+----------------+
                 |                                 |
       +---------v---------+             +---------v---------+
       | Lambda Consumer   |             | Kubernetes Worker |
       | Event Source Map  |             | long poll + pool  |
       +---------+---------+             +---------+---------+
                 |                                 |
                 +----------------+----------------+
                                  |
                         +--------v---------+
                         | Processing Core  |
                         | schemas/validate |
                         | persist/decide   |
                         +---+----------+---+
                             |          |
                    +--------v--+   +---v---------+
                    | DynamoDB  |   | DLQ adapter |
                    +-----------+   +-------------+
```

The two input adapters translate their runtime-specific delivery contracts into the same processing operation. The core must not depend on Lambda event types, Kubernetes APIs, or SQS deletion mechanics.

## Shared Processing Core

The shared core will contain:

- Configuration needed to construct AWS clients and processing dependencies.
- Schema catalog loading from DynamoDB.
- A local schema cache with a configurable TTL and a default of five minutes.
- Single-flight schema refresh so concurrent messages do not trigger duplicate scans.
- JSON Schema validation.
- DynamoDB persistence and idempotency behavior.
- The existing processing outcome model: `ACK`, `RETRY`, and `TO_DLQ`.

The schema cache behavior is the same in both runtimes:

1. Load and compile the schema catalog during startup.
2. Check cache freshness before processing.
3. When expired, allow one refresh while other processing waits.
4. Continue using the previous valid cache if a refresh fails, log the failure, and retry on a later refresh attempt.
5. Fail startup if no valid schema cache can be loaded.

The cache is local to a process. It is not shared between pods or Lambda execution environments.

## Kubernetes Worker Behavior

The worker will run as a long-lived process in a Kubernetes `Deployment`.

### Polling

- Use SQS long polling, with the wait time and batch size configurable.
- Poll only while the process is accepting work.
- Dispatch messages to a bounded processing pool.
- Do not create an unbounded goroutine per received message.

### Message outcomes

- `ACK`: delete the source SQS message after successful persistence.
- `RETRY`: do not delete the source message; allow visibility to expire or explicitly change visibility according to the implementation.
- `TO_DLQ`: send the message to the configured DLQ, confirm the send succeeded, then delete the source message.
- If sending to the DLQ fails, do not delete the source message.

### Visibility and duplicates

The current queue visibility timeout is 30 seconds. Worker configuration and processing limits must ensure that a message does not become visible while it is still being processed. For work that can exceed the initial timeout, the worker must extend visibility periodically. At-least-once delivery remains the contract, so duplicate delivery must remain safe through idempotent persistence.

### Graceful shutdown

On `SIGTERM` or `SIGINT`:

1. Stop polling for new messages.
2. Stop accepting work into the processing pool.
3. Allow in-flight work to finish until the configured shutdown deadline.
4. Delete only messages that reached `ACK` or successfully completed `TO_DLQ` handling.
5. Leave unfinished messages undeleted so SQS can redeliver them.
6. Exit with a clear status for Kubernetes.

## Configuration

The worker will receive configuration through environment variables, with Secrets and ConfigMaps supplied by Kubernetes. At minimum:

- AWS region and credential/pod-identity settings.
- Source queue URL and DLQ URL.
- Schema table name and events table name.
- Optional AWS endpoint override for LocalStack.
- SQS long-poll wait time and receive batch size.
- Worker concurrency.
- Visibility extension interval and shutdown deadline.
- Schema cache TTL, defaulting to five minutes.

No static AWS credentials should be stored in Kubernetes Secrets. The deployment should use IRSA or the cluster's equivalent pod identity mechanism.

## Kubernetes Resources

The implementation should add:

- An application Dockerfile producing a minimal non-root image.
- A `Deployment` for the worker.
- Service account and AWS pod-identity configuration.
- ConfigMap/Secret references for runtime configuration.
- Readiness and liveness behavior appropriate for a queue consumer.
- Resource requests and limits.
- Optional HPA or KEDA scaling based on SQS backlog and message age.

The first rollout may use one replica, but the process must be safe to run with multiple replicas.

## Observability

The worker will keep structured JSON logs on stdout so the platform can collect them. Logs should include message identifiers, tenant identifier, event type, outcome, retry/DLQ reason, processing duration, and consumer runtime.

Worker metrics should cover at least:

- Messages received, acknowledged, retried, and sent to the DLQ.
- Processing failures and delete failures.
- In-flight work and processing duration.
- Schema cache loads, refreshes, refresh failures, and cache age.
- Polling errors and visibility-extension failures.

Lambda-specific X-Ray integration remains in the Lambda adapter. The processing core must not require X-Ray. Kubernetes tracing can be added independently later.

## Security and Permissions

The Kubernetes identity needs only the permissions required for this flow:

- Receive, delete, and change visibility on the source queue.
- Send messages to the DLQ.
- Read schemas from DynamoDB.
- Write events to DynamoDB.
- Emit logs and metrics through the cluster's platform integration, if required.

Permissions should be separate from the Lambda execution role even when the allowed AWS actions are similar.

## Testing Strategy

Add tests for:

- Worker message translation and outcome handling.
- Delete behavior for `ACK`, `RETRY`, and `TO_DLQ`.
- DLQ send failure preserving the source message.
- Bounded worker concurrency.
- Visibility extension.
- Polling cancellation and graceful shutdown.
- Schema cache TTL, single-flight refresh, stale-cache fallback, and startup failure with no valid cache.
- Lambda behavior remaining compatible with `ReportBatchItemFailures`.
- LocalStack integration for the end-to-end worker path.

## Rollout and Migration

1. Extract shared initialization and processing behavior without changing Lambda semantics.
2. Add the worker executable, unit tests, and local integration path.
3. Build and deploy the worker with one replica while Lambda remains enabled.
4. Observe queue age, duplicate rates, retry/DLQ behavior, cache refreshes, and DynamoDB capacity.
5. Increase worker replicas or enable queue-driven autoscaling if required.
6. Disable the Lambda Event Source Mapping only after Kubernetes meets the operational objectives.
7. Keep the Lambda entrypoint available for rollback until the Kubernetes-only mode is proven.

## Open Implementation Questions

The following are intentionally implementation-level and should be resolved during delivery:

- Exact SQS receive batch size and worker concurrency per environment.
- Visibility extension interval and maximum processing duration.
- Kubernetes health-check details for the selected deployment platform.
- HPA versus KEDA, depending on the cluster's standard tooling.
- Exact metrics backend and alert thresholds.
