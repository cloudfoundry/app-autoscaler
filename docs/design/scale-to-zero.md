# Design: Scale-to-Zero / Scale-from-Zero (Activator)

Status: Proof-of-Concept design
Branch: `scale-to-zero`
Author: (draft, assisted)
Date: 2026-08-31

## 1. Summary

Today app-autoscaler enforces a floor of **1** instance (`instance_min_count >= 1`). This
proposal lets an app scale down to **0** instances, and — critically — scale **back up from 0**
on the first incoming HTTP request, so a client sees (at worst) a slower request rather than a
`503`/`404`.

The wake-on-request path is handled by a **new microservice, the `activator`**. The name follows
[Knative's `activator`](https://knative.dev), which is precisely a cold-start proxy that steps
into the request path only while a workload is scaled to zero — a closer fit than KEDA's
"interceptor" (which is *always* in the data path).

The three PoC deliverables (per product decision) are:

1. **Relax validation** to allow `instance_min_count: 0`.
2. **Scale-to-zero path in the scaling engine** — treat a scale to `0` as a special case that
   parks the app behind the activator.
3. **The new `activator` microservice** — parks the app's routes behind itself while at zero,
   holds the first request, wakes the app, waits for readiness, forwards, and un-parks.

Scale-from-zero via a schedule (the "2.1" case) works out of the box once validation is relaxed,
and is captured for free by the activator's unpark loop (see §5.3).

## 2. Background: CF routing, and why the 2019 approach doesn't port

### 2.1 How CF routing makes this different from KEDA

- The **Gorouter** routes directly to app instances; the **route-emitter** on each Diego cell
  registers/deregisters an app's instance endpoints as they come and go.
- When an app is scaled to **0**, its route mapping still exists but the endpoint pool is empty.
  Gorouter returns `503`/`404`.
- To wake on request, *something* reachable by Gorouter must occupy the request path for that
  route while the app is at zero.

### 2.2 Historical note: the BOSH-deployed approach (CFEU 2019)

A prior "Scale to Zero" design (presented at CF EU 2019, with team involvement) worked because
the autoscaler was **BOSH-deployed**: the interceptor ran on a BOSH VM with a **stable,
Gorouter-reachable IP**. That made **direct `routing-api` `ip:port` registration** the right
tool — register `<vm-ip>:<port>` as the backend for the parked route, and Gorouter reaches it
directly.

The autoscaler is now a **CF app** (MTA-deployed). That stable IP is gone:

- The container overlay IP is ephemeral and per-instance; it changes on restart/rebalance and
  cannot be reliably held across a ~120s route TTL, nor shared across multiple activator
  instances.
- Registering the ingress/Gorouter IP would create a routing loop and not reach us.

So the deployment-model change is the reason the mechanism must change. A CF app's only stable,
Gorouter-reachable handle is **its own route** — which is exactly what a **route service**
consumes. This design therefore replaces `ip:port` registration with **route-service binding**.
(`ip:port` registration is documented as the rejected alternative in §8.)

### 2.3 Route services (the chosen mechanism)

Binding a `route_service_url` to a route makes Gorouter forward matching requests **to the route
service first**, before the app:

1. Client → Gorouter → **route service** (over HTTPS), with headers `X-CF-Forwarded-Url`
   (original URL), `X-CF-Proxy-Signature`, `X-CF-Proxy-Metadata`.
2. Route service does its work, then forwards **back to `X-CF-Forwarded-Url`** carrying those
   three headers unchanged.
3. Gorouter recognizes the proxy signature → **bypasses the route service** → delivers to the
   app instances. This is how the loop is broken by design.

The `route_service_url` is a normal HTTPS URL, so it can be **the activator's own public route**
(`https://autoscaler-activator.<sys-domain>`). This solves the reachability problem: the
activator is reached the normal CF way, via its route — no self-IP discovery.

**Confirmed:** Gorouter evaluates the route-service binding **before** endpoint selection, as long
as the route mapping still exists. Scaling to 0 empties the endpoint pool but does **not** remove
the route mapping, so a request to a parked (0-instance) app **is** forwarded to the route
service. This is the behavior scale-from-zero depends on.

## 3. Interception mechanism: bind only while parked

The route service is a single, long-lived **user-provided service instance** (UPSI) created once
at deploy time:

```
cf create-user-provided-service autoscaler-activator-rs -r https://autoscaler-activator.<sys-domain>
```

The UPSI is never recreated per app — only **bindings** to it churn, via the CF v3 API on the
**route** (not the app):

- Bind:   `POST /v3/service_route_bindings` (route guid + UPSI guid). **Async** (`202` + job).
- Unbind: `DELETE /v3/service_route_bindings/:guid`. **Async**.
- List:   `GET /v3/service_route_bindings` (used for reconcile, §5.4).

Bind/unbind cycles on the same route are supported and repeatable.

**Why bind-only-while-parked** (vs binding persistently): zero extra latency hop when the app is
running. The cost is per-park/wake async CF calls that must be sequenced against the scale (§5).
If async bind timing proves troublesome in the PoC, the documented fallback is **bind
persistently** for scale-to-zero-enabled apps (one always-on activator hop; the model Knative/KEDA
accept). This is a config choice, not a redesign.

## 4. Component overview

```
   PARK (scale-to-zero):
   trigger(target 0) ─▶ scalingengine ─┐
                                        │ 1. bind route service for app's routes (async, poll to done)
                                        │ 2. only then Processes.Scale(app → 0)
                                        └ (route mapping stays; endpoint pool empties)

   WAKE (scale-from-zero) — two decoupled loops in the activator:

   Loop A (request path / route service):
     client ─▶ Gorouter ─▶ activator ─┐  hold request
                                       │  POST /v1/apps/{appid}/scale (→1) ─▶ scalingengine ─▶ Processes.Scale(1)
                                       │  wait for app's Upsert  ◀── (shared signal from Loop B feed)
                                       └  forward held request to X-CF-Forwarded-Url (+ proxy headers) ─▶ Gorouter ─▶ app

   Loop B (unpark, driven by routing-api Upsert events):
     /routing/v1/events ── Upsert(appId) ─▶ if app is parked: release held requests, then unbind route service
                                            (fires for ANY cause of app-up: request, schedule, manual cf scale)
```

## 5. Detailed flows

### 5.1 Park (scale-to-zero) — bind BEFORE scaling down

1. A trigger to scale to 0 reaches the scaling engine (dynamic rule clamped to
   `instance_min_count = 0`, or a schedule with `instance_min_count: 0`).
2. Engine runs its normal checks (lock, cooldown, `ComputeNewInstances` clamping).
3. **New:** before the final scale, the app's routes are bound to the activator route-service UPSI
   (`POST /v3/service_route_bindings` per route; poll the async job(s) to completion).
4. **Only after bindings are active**, `ScaleAppWebProcess(appId, 0)`.

Ordering rationale: bind-first closes the atomicity gap. If we scaled to 0 first, a request in the
window between "0 instances" and "binding active" would hit an empty pool → 503. Because the route
mapping persists at zero and Gorouter checks the binding first, bind-then-scale gives continuous
coverage.

(Design point: whether the bind is issued by the scaling engine or delegated to the activator via
a notification is an implementation choice — see §7.2. Either way the *ordering* is bind → confirm
→ scale-to-0.)

### 5.2 Wake — Loop A (request handling via route service)

1. Request for a parked route → Gorouter → activator, with `X-CF-Forwarded-Url` /
   `X-CF-Proxy-Signature` / `X-CF-Proxy-Metadata`.
2. Activator identifies the target app (from the parked-route registry / forwarded URL) and
   **holds** the request (connection open).
3. Activator `POST`s to scaling engine `/v1/apps/{appid}/scale` with a trigger yielding target `1`
   (reuses the existing endpoint + `models.Trigger`; no new engine endpoint).
4. Activator waits for the app's readiness signal: the `Upsert` for its real endpoint on
   `/routing/v1/events` (route-emitter registers an instance only once CF deems it healthy — we
   require apps to define a correct readiness health check).
5. On `Upsert`, activator forwards the held request back to `X-CF-Forwarded-Url` **with the three
   proxy headers**; Gorouter bypasses the route service and delivers to the live app. Response
   streams back to the client.
6. **Timeout:** if no `Upsert` arrives within a configurable readiness timeout (e.g. app
   crash-loops on startup), the activator returns **`503` + `Retry-After`** to the client. The
   client may retry; a background scale-up may still complete.

### 5.3 Wake — Loop B (unpark, decoupled, Upsert-driven)

Loop B is **independent of Loop A**. It consumes `/routing/v1/events` and, on an `Upsert` for an
app currently parked, runs the unpark sequence: release/complete any held requests for that app
(the shared signal Loop A waits on), then **unbind** the route service for that app's routes
(lazily — after any forward, off the critical path).

Because Loop B keys on the route registration itself — not on "a held request caused this" — the
`Upsert` is the **single source of truth for "app is up → unpark it."** This uniformly captures:

- **Request-driven wake** (Loop A initiated the scale-up).
- **Manual `cf scale` / `cf start` from zero** — app comes up out-of-band, `Upsert` arrives, Loop B
  unbinds. No held request, no scaling-engine involvement; the app does not get stuck behind the
  activator hop.
- **Schedule-driven wake** — engine scales up on schedule; same `Upsert` path.

`Upsert` for a non-parked app is a no-op (idempotent). Un-parked apps that later idle back to zero
are re-parked normally via §5.1.

### 5.4 Safety net — Upsert + periodic reconcile

The SSE event stream can be missed (activator restart mid-park, dropped connection, event gap). In
addition to Upsert-driven unpark, Loop B **periodically reconciles** parked-app state against CF:
for each app the activator believes is parked, check actual instance count / route bindings
(`GetAppAndProcesses`, `GET /v3/service_route_bindings`). If an app is actually up (or its instance
count > 0) but still shows parked, run the unpark sequence. This also lets a freshly (re)started
activator rebuild its parked-app registry from CF rather than trusting only in-memory state.

## 6. Validation change (deliverable 1)

Single logical change, in two byte-identical schema files:

- `api/policyvalidator/json-schema/v0.1/scaling-policy.schema.json:23`
- `api/broker/binding_request_parser/v0_1/scaling-policy.schema.json:23`

Change `instance_min_count` `"minimum": 1` → `"minimum": 0`; keep it `required`. The Go validator
(`api/policyvalidator/policy_validator.go`) enforces only relational rules (`min <= max`, `initial`
within bounds), which stay correct at a floor of 0. Schedules already permit `0`. Add tests:
`min_count: 0` accepted; `max_count: 0` still rejected (an app that can never serve).

## 7. New component scaffolding (deliverable 3)

Repo root is the Go module `code.cloudfoundry.org/app-autoscaler/src/autoscaler`; each component is
a top-level dir. The `metricsforwarder`/`eventgenerator` services are the templates.

- Code: `activator/cmd/activator/main.go` (use `startup.Bootstrap`), `activator/config/`,
  `activator/server/` (the route-service reverse proxy), plus `activator/default_config.json`,
  `activator/exampleconfig/config.yml`, `activator/security-group.json`.
- Build: automatic once `activator/cmd/activator/main.go` exists (root `Makefile` auto-discovers
  `main.go` dirs). Add `activator` to `MODULES` if other tooling needs it.
- Deploy (MTA): add a `type: go` module block + `activator-config` user-provided-service in
  `mta.tpl.yaml`; route/config/secret overrides in `scripts/extension-file.tpl.yaml`. Additionally,
  the **route-service UPSI** (`autoscaler-activator-rs`) must be created and pointed at the
  activator's public route at deploy time.
- Inter-service calls: reuse `helpers.CreateHTTPSClient` + `models.TLSCerts` (CF instance identity
  certs) + the shared `routes/` table — exactly as eventgenerator→scalingengine
  (`eventgenerator/generator/evaluator.go:192`).

### 7.1 CF API & routing-api access, credentials

Target platform: **OSS cf-deployment**. Two external dependencies:

- **CF v3 API** — to bind/unbind service route bindings and read app/route state. The pinned
  `go-cfclient/v3` already provides route + service-route-binding clients; surface the needed calls
  through the existing `cf` wrapper (which today exposes no route methods).
- **routing-api** — `GET /routing/v1/events` (SSE), scope `routing.routes.read`. Official Go client
  `code.cloudfoundry.org/routing-api` (`SubscribeToEvents`). **Note:** with the route-service
  mechanism we likely need only `routing.routes.read` (readiness signal), *not*
  `routing.routes.write` — a smaller privilege footprint than the old `ip:port` design.
- **Reachability probe (prerequisite):** confirm `routing-api` is reachable via Gorouter; if not,
  open an application security group for egress (we already do this for log-cache —
  `metricsforwarder/security-group.json`).
- **Credentials:** UAA client with `routing.routes.read`, secret pulled from **credhub at build
  time** (`scripts/build-extension-file.sh` via `credhub interpolate`) → `activator-config` UPSI →
  read at runtime as VCAP creds. Mirrors the eventgenerator log-cache UAA plumbing
  (`scripts/extension-file.tpl.yaml:124`, `models.UAACreds`).

### 7.2 Who issues the bind: engine vs activator

Two implementation options for §5.1 step 3:
- **Engine binds** synchronously before scaling to 0 (keeps ordering trivially correct; engine
  gains CF route-binding capability).
- **Engine notifies activator** to park, activator binds and confirms, then engine scales
  (keeps route logic in one place; adds a handshake).
PoC can start with whichever is simpler; the ordering contract (bind → confirm → scale-0) is the
invariant.

## 8. Rejected alternative: direct `routing-api` ip:port registration

Register the activator's `ip:port` as the parked route's backend via
`POST /routing/v1/routes` (TTL-refreshed), unregister on wake. This was the 2019 BOSH design and is
rejected now only because a CF-app activator has no stable, Gorouter-reachable IP (§2.2). Kept here
because if the activator were ever run on a stable-IP substrate again, it becomes viable — and it
avoids the always-possible route-service latency hop.

## 9. Open questions / risks to resolve during PoC

1. **Route-service bind/unbind latency** — measure async job completion time; confirm park isn't
   sluggish and woken apps don't linger behind the hop. (Bind is on the park critical path; unbind
   is lazy/off-path.)
2. **Cooldown vs wake** — the engine's DB cooldown could suppress a cold-start scale-up; the wake
   path likely needs a bypass or distinct path so cold starts aren't treated as oscillation.
3. **Activator HA** — multiple activator instances: the held request lives on the instance Gorouter
   chose; any instance must be able to complete the forward (route-service forward goes back through
   Gorouter, so this is naturally fine). Parked-app registry should be reconstructable via reconcile
   (§5.4) rather than shared in-memory state.
4. **Route-service "Experimental" edges** — validate zero-instance forwarding and header handling on
   the actual target deployment.
5. **Multiple / path-based routes per app** — bind all of an app's routes; handle path routes.
6. **Non-HTTP / no-route apps** — PoC scopes to HTTP routes. Apps with no routes: scale-to-zero is
   fine but only a schedule can wake them (no request trigger).
7. **Security** — activator forwards arbitrary app traffic; lock down (mTLS to engine, scoped UAA
   creds, ASG egress only) and forward strictly per `X-CF-Forwarded-Url`.
8. **Re-park loop stability** — avoid flapping between park and wake for bursty-but-idle apps
   (respect cooldown / a minimum-parked dwell time).

## 10. Out of scope for PoC

Metrics/observability polish, activator self-autoscaling, TCP routes, request-buffering
backpressure limits, and production hardening of the bind/scale atomicity beyond the documented
ordering.
