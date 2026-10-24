# Concurrent Consumers on One SQS Queue

Status: accepted

Lambda and the Kubernetes Worker will consume the same SQS queue concurrently during the migration period. SQS distributes messages between competing consumers, so each message is expected to be processed by one consumer attempt rather than delivered independently to both runtimes. This allows Kubernetes capacity to be introduced without creating a second event stream.

## Message Distribution

```text
                         SQS queue
                 +-----------+-----------+
                 |                       |
          message A, B, C          message D, E
                 |                       |
          Lambda polling          Worker polling
                 |                       |
          A and C processed       B, D and E processed
```

The exact distribution is controlled by SQS and can change between receives. The system must not depend on a message being assigned to a particular runtime.

## Important Clarification

This is a competing-consumer model, not fan-out:

```text
One source message -> one consumer attempt at a time
                    Lambda OR Kubernetes
```

A message can still be delivered again after visibility expires, a consumer crashes, or a delete fails. That is a redelivery, not an intentional copy for the second runtime. Idempotency is covered by ADR 0003.

## Rollout Example

| Phase | Lambda | Kubernetes | Expected purpose |
| --- | --- | --- | --- |
| 1 | Enabled | Disabled | Existing baseline |
| 2 | Enabled | 1 replica | Validate worker with low blast radius |
| 3 | Enabled | N replicas | Measure mixed capacity and behavior |
| 4 | Disabled | N replicas | Kubernetes-only operation |
| 5 | Available for rollback | N replicas | Keep code/deployment path until stable |

During phase 2 or 3, queue depth is a property of the combined consumers. Comparing Lambda metrics alone with total queue depth will be misleading.

## Considered Options

- Give Lambda and Kubernetes separate queues: rejected because both runtimes would then process every event and require a fan-out or deduplication design.
- Disable Lambda before deploying Kubernetes: rejected because it removes the safest incremental rollout and rollback path.

## Consequences

Capacity, retry behavior, and observability must be evaluated across both runtimes while they coexist.

The two consumers must use the same queue URL, message attributes, schema catalog, event table, and DLQ contract. A separate queue would change the semantics from competition to fan-out and is outside this decision.

## Implementation Checklist

- [ ] Configure both runtimes with the same source queue URL.
- [ ] Confirm the worker uses the same message attributes: `client_id`, `event_id`, and `event_type`.
- [ ] Track consumer identity in logs and metrics so Lambda and Kubernetes can be distinguished.
- [ ] Test messages while both consumers are active and verify no intentional double processing.
- [ ] Document the Lambda Event Source Mapping as the switch used for the final cutover.

See also: [ADR 0003](0003-at-least-once-delivery-and-idempotent-persistence.md) and [ADR 0006](0006-phased-lambda-to-kubernetes-migration.md).
