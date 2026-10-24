# TTL Schema Cache with Single-Flight Refresh

Status: accepted

Both runtimes will use the same local compiled schema cache with a configurable TTL, defaulting to five minutes. The cache loads during startup, refreshes through one in-flight refresh operation when expired, and keeps the previous valid cache when a refresh fails. A process with no valid initial cache fails startup because it cannot safely validate events.

## Cache Lifecycle

```text
STARTING
   |
   +--> load catalog and compile schemas
          |
          +--> success --> FRESH cache --> process messages
          |
          +--> failure --> process exits; no unsafe validation

FRESH cache -- TTL expires --> REFRESHING
                                  |
                                  +--> success --> replace atomically -> FRESH
                                  |
                                  +--> failure --> keep previous cache
                                                   retry later
```

The cache is replaced only after the complete catalog has been loaded and compiled successfully. A partially loaded or partially compiled catalog must never become visible to processors.

## Refresh Example

With the default five-minute TTL:

```text
12:00  cache loaded with invoice.created v1
12:05  first message notices expiration
12:05  one refresh starts; concurrent messages wait
12:05  refresh succeeds; all messages use the new cache
```

If the DynamoDB scan fails at 12:05:

```text
12:05  refresh fails
12:05  previous valid cache remains available
12:06  next eligible check retries refresh
```

This fallback avoids turning a temporary catalog-read outage into an immediate total consumer outage, while metrics and logs make the stale cache visible.

## Single-Flight Requirement

For `N` concurrent messages observing an expired cache:

```text
Expected: N messages -> 1 DynamoDB scan -> N messages continue
Wrong:    N messages -> N DynamoDB scans
```

The refresh lock/single-flight mechanism must also publish the refresh error to waiting callers. It must not leave callers blocked indefinitely if compilation or loading fails.

## Runtime Parity

Lambda execution environments may be reused across invocations, while Kubernetes pods are long-lived. The same TTL check must therefore run before processing in both runtimes. A Lambda cold start and a Kubernetes pod start both perform the initial load; a reused Lambda environment must not bypass the TTL forever.

## Considered Options

- Load schemas for every event: rejected because it adds DynamoDB latency and cost to the hot path.
- Use a process-lifetime cache with no refresh: rejected because schema changes would require restarting every consumer environment.
- Let every concurrent message refresh independently: rejected because it creates duplicate DynamoDB scans and inconsistent refresh behavior.

## Consequences

Schema changes become visible within the TTL plus refresh latency, and refresh failures must be observable. The cache is process-local; it is not a distributed schema registry or cross-pod cache.

The five-minute value is a default, not a hard domain guarantee. Environments with faster schema rollout requirements may lower it; environments prioritizing lower DynamoDB read load may increase it.

## Implementation Checklist

- [ ] Define a shared schema-cache component used by both entrypoints.
- [ ] Make TTL configurable and default it to five minutes.
- [ ] Compile the complete replacement before swapping the active cache.
- [ ] Serialize or single-flight refresh operations.
- [ ] Preserve the previous valid cache on refresh failure.
- [ ] Fail startup when the initial load cannot produce a valid cache.
- [ ] Record load duration, refresh result, cache age, and refresh errors.
- [ ] Test concurrent expiration with one catalog read.

See also: [ADR 0001](0001-separate-runtime-entrypoints-shared-processing-core.md).
