# Shared application-state observation

## Ownership

`appstateobserver` is a CF application deployed once logically per foundation.
Its replicas share a PostgreSQL journal. It observes CF; it does not scale apps,
interpret scaling bounds, or register activator backends.

```text
Cloud Controller audit journal
             |
      bulk window reader <---- policy DB membership
             |
     durable invalidations
             |
   coalesced CF state refresh
             |
   authenticated compacted change feed
        /                  \
scaling engine          activator
local no-op cache       parking reconciler
        |                  |
fresh CF read + scale   routing NATS + readiness
```

- CF owns desired app state and desired web instance counts.
- The observer owns verified observations, invalidations, and reader progress.
- The scaling engine owns policy/schedule evaluation and CF mutations.
- The activator owns its backend registrations and readiness decisions.
- Zero desired instances, zero healthy backends, STOPPED, and parked are distinct.

## Observation and delivery

`cf.AuditEventSource` is separate from the existing `cf.CFClient` interface.
It uses the authenticated go-cfclient transport, token renewal and pagination.
Relevant app events include scaling, process changes, start/stop, app updates,
deletion and route mapping changes. Route-target updates/deletions conservatively
invalidate all managed apps because these events need not contain an app GUID.

The reader fetches bounded timestamp windows, with a one-minute overlap and a
two-second visibility delay. GUID deduplication handles identical timestamps and
replayed windows. The checkpoint does not advance on an incomplete/failed window.
The default window is one minute, capped at 20 pages of 1,000 events. An overloaded
window fails visibly rather than silently dropping events; reduce `window` if
necessary. Delays beyond the overlap and event-retention gaps are repaired by
state reconciliation, not assumed impossible.

A transaction-scoped PostgreSQL advisory lock elects the active reader. Its
checkpoint, membership changes, deduplication records and invalidations commit
together. Connection loss rolls back and releases leadership. A failed policy
membership read never looks like mass unbinding.

Refresh runs independently, with at most four concurrent CF reads per replica
and batches of twenty dirty/expired observations. Repeated app events coalesce.
Workers claim jobs with `SKIP LOCKED` and a ninety-second recovery lease, avoiding
duplicate reads across replicas while allowing recovery after worker loss.
Reads capture a revision before calling CF; results cannot overwrite a later
invalidation or membership change. Failed reads retry after fifteen seconds.
Default background refresh is five minutes, rather than one read per trigger.

`appstate` defines the consumer contract:

- `GET /v1/changes?after=<revision>`: up to 500 latest observations, cursor,
  pagination flag, and foundation-reader checkpoint (`checked_at`).
- `POST /v1/apps/<guid>/invalidate`: signal an autoscaler mutation.

Both surfaces use the existing Gorouter/XFCC authentication with autoscaler
org/space identity validation. Consumer connections use instance identity TLS.
There is no new general-purpose NATS bus or message broker.

The feed is **compacted state**, not an immutable history of every transition.
Both dirty and refreshed revisions invalidate scaling caches. Revisions are
allocated under a shared metadata-row lock, so cursor ordering follows commit
ordering. Each consumer starts at zero to reconstruct current state; cursor
persistence is unnecessary for these idempotent consumers. Unbinding tombstones
remain recoverable. Deduplication records are pruned after 24 hours.
If a consumer cursor is ahead of a restored database, the feed requests a reset
and returns the current snapshot; local suppression state is discarded.

## Scaling engine / APPAUTOSCALER-132

The optional `WithAppState` seam enables a local bounded cache. Only an earlier
successful CF read can populate it; observer payloads cannot directly authorize
scaling. The cache has a fixed, non-sliding lifetime (default 30 seconds), and is
usable only after its consumer has drained the change feed and the foundation
reader is fresh. Failed feed reads clear the cache.

For each trigger, the engine reads current policy or active-schedule bounds and
computes/clamps the target. If fresh cached state proves target equals current
count, it records an ignored decision without calling CF. Otherwise it executes
the existing fresh-CF path, including STARTED/disable-label checks and cooldown.
Wake triggers always bypass this optimization.

Invalidation tokens fence overlapping reads. Dynamic and scheduled writes clear
the local cache and notify the observer before/after their mutation. Notifications
are best effort: failure cannot prevent scaling. Other replicas invalidate on
their next feed poll; expiry bounds the effect of missed/delayed notifications.

This reduces no-op **CF reads** at both instance limits. It does not suppress
history writes or implement quota-aware partial scaling. Quota-failure backoff
remains separate because another app or a quota edit can release capacity.

## Activator

The reconciler consumes the whole feed on every activator replica. Each replica
registers/unregisters its own backend, so reconciliation must not elect a single
activator worker. Peer activator registrations are filtered by the shared app
GUID from `VCAP_APPLICATION`, in addition to the existing instance identity filter.

Every actionable observation is rechecked against current CF state and policy:

- Managed, wake enabled (`instance_min_count: 0`), STARTED, not disabled, desired
  web count zero: ensure parking and reconcile route registrations.
- STOPPED, disabled, deleted, unbound, or wake disabled: remove local parking.
- Desired count positive: keep parking until a real routing backend is ready.

The NATS readiness watcher checks current desired state before accepting a
backend, preventing a retiring backend from immediately undoing manual-zero
parking. Failed unpark does not release held requests. Request waiters remain
registered before triggering wake.

Successful reconciliation is idempotent. Per-app failures stay pending and do
not prevent other apps from reconciling. A slower one-minute safety pass handles
policy edits/registry loss for known zero or parked apps. Normal five-second feed
polls do not fetch every parked app. On restart, the persisted observer snapshot
reconstructs parking through current CF routes. Removed route registrations are
explicitly withdrawn when routes change.

The registry is still local memory; an activator restart reconstructs it rather
than loading an authoritative parking flag. Registrations from a dead instance
expire through the existing routing TTL. This is not exactly-once delivery or an
atomic transaction across CF, PostgreSQL and routing NATS.

The engine retains synchronous **park-before-scale** for its zero mutations.
Manual `cf scale -i 0` is necessarily repaired after the fact: requests can fail
between the external scale and observer-driven parking. No event reader closes
that gap atomically.

## Deployment and operation

The MTA includes the observer, configuration service and migration task.
Default deployment: two observer replicas, five-second journal/consumer polling,
one-minute windows, five-minute observation refresh. The PoC uses observer-owned
`appstate_*` tables in the existing `scalingengine_db`, avoiding a new database
provisioning dependency. Only the observer accesses these tables.

`APPSTATEOBSERVER_HOST` and `APPSTATEOBSERVER_INSTANCES` override deployment
defaults. Scaling engine and activator opt in through `app_state.url`; omit it to
retain their previous standalone behaviour. The activator now requires the policy
database binding when observation is enabled. The observer CF identity must be
able to read audit events and app/process/routes for every managed app; deploying
it does not grant those permissions automatically.

Reader failures are logged as `audit-reader.poll-failed-retrying`; CF refresh
failures include `appID`. Consumers fail open (scaling uses CF), or retain current
parking while observations are unavailable. Monitor the `checked_at` lag; a
running HTTP service alone does not imply a healthy journal reader. The health
server exports `autoscaler_app_state_reader_lag_seconds` and a journal-readiness
check, independently of HTTP liveness.

## Tests

- Unit tests: fixed cache expiry, reader failure/lag, invalidation fencing,
  min/max no-op CF request counts, opposite-direction and wake triggers,
  policy/schedule changes, current-state parking and STOPPED/disabled cleanup.
- CF transport tests: time filters, bearer authentication, pagination and cap.
- PostgreSQL tests execute the actual migration in isolated temporary schemas:
  transaction rollback, reader election, GUID deduplication, revision fencing,
  compacted pagination, tombstones and manual zero/up refresh.
- Acceptance: policies with a rule for an unreported custom metric exercise manual zero with request-driven wake
  and manual zero followed by manual scale to two, verifying unpark before any
  HTTP request. Authenticated foundation-admin log access verifies routing-side
  parking without adding a public test-only endpoint.

```sh
APPSTATE_TEST_DB_URL='postgres://postgres:postgres@localhost/postgres?sslmode=disable' \
  go test ./appstateobserver/...
go test ./appstate/... ./scalingengine ./activator ./cf
SUITES=scaletozero make mta-acceptance-tests
```

Acceptance uses `activator_app_name` (default `activator`, the MTA module's app
name) and requires a deployed observer with the migration applied. No acceptance deployment is
performed by unit/integration tests.
