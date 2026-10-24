# Phased Lambda to Kubernetes Migration

Status: accepted

The migration will begin with Lambda and Kubernetes consuming concurrently, then move to Kubernetes-only operation by disabling the Lambda Event Source Mapping after the worker meets operational objectives. The Lambda entrypoint remains available during the transition for rollback. This path provides incremental validation without requiring a cutover that removes the existing processing path before the replacement is proven.

## Migration Phases

```text
Phase 0       Phase 1          Phase 2             Phase 3
Lambda only -> Lambda + 1 pod -> Lambda + N pods -> Kubernetes only
 baseline       canary           scale validation    final mode
```

### Phase 0: Baseline

Record current queue age, throughput, Lambda errors, retry counts, DLQ volume, DynamoDB throttling, and processing latency. These measurements are the comparison point for the worker.

### Phase 1: Canary Worker

Deploy one Kubernetes replica against the same source queue. Keep Lambda enabled. Verify:

- Worker can load and refresh schemas.
- Worker processes valid messages and deletes them.
- Recoverable failures remain retryable.
- Terminal failures reach the DLQ before source deletion.
- Worker shutdown does not delete unfinished messages.

### Phase 2: Scale Validation

Increase replicas or enable queue-driven autoscaling. Evaluate the combined Lambda and worker fleet, not either consumer in isolation. Confirm DynamoDB capacity and SQS visibility settings remain safe.

### Phase 3: Kubernetes-Only

Disable the Lambda Event Source Mapping, observe the queue drain and worker health, and retain the Lambda deployment configuration for rollback. The Lambda code may remain in the repository even when its production trigger is disabled.

## Go/No-Go Criteria

Proceed to the next phase only when:

- No unexplained message loss is observed.
- No unexplained growth in duplicate persistence attempts occurs.
- Queue age remains within the service objective under expected load.
- `RETRY` and `TO_DLQ` behavior matches Lambda behavior.
- Graceful shutdown and pod restart tests pass.
- Schema cache refresh and failure behavior are observable.
- Required AWS permissions work through pod identity.

## Rollback

If the worker causes unacceptable errors or queue age:

1. Stop or scale down the Kubernetes Deployment.
2. Re-enable the Lambda Event Source Mapping.
3. Confirm Lambda resumes consuming the same queue.
4. Investigate unprocessed, retried, and DLQ messages before another canary.

Because the source queue and processing core remain unchanged, rollback does not require moving messages between queues or restoring a second data store.

## Considered Options

- Immediate Kubernetes cutover: rejected because it concentrates deployment, scaling, and message-handling risk into one change.
- Permanent dual operation: rejected because it retains unnecessary operational cost and makes ownership of the primary runtime ambiguous.

## Consequences

The rollout must define measurable readiness criteria, monitor both consumers during coexistence, and preserve compatible message semantics across the entire migration.

The coexistence window temporarily increases operational surface area and can increase total consumer capacity beyond the steady-state requirement. That cost is accepted in exchange for a reversible rollout.

## Implementation Checklist

- [ ] Define baseline dashboards and alert thresholds before the canary.
- [ ] Deploy the worker with one replica and Lambda still enabled.
- [ ] Validate message outcomes and shutdown behavior in a shared queue.
- [ ] Define the exact operation that disables/enables the Lambda Event Source Mapping.
- [ ] Document the rollback owner and maximum acceptable queue age.
- [ ] Record the decision to enter Kubernetes-only operation.

See also: [ADR 0002](0002-concurrent-consumers-on-one-sqs-queue.md) and [RFC 0001](../rfc/0001-lambda-and-kubernetes-consumers.md).
