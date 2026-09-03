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

**⚠️ DISPROVEN BY LIVE VALIDATION (2026-09-02).** The assumption above — that a route survives
scale-to-zero and a bound route service still receives traffic — is **false on standard CF**. See
§2.1.

## 2.1 Findings from live validation (blocking) — READ BEFORE IMPLEMENTING

The route-service interception design was implemented and exercised end-to-end against a
bosh-deployed cf-deployment foundation (`autoscaler-mta-*`). Results:

- **Scale-to-zero works up to the last step:** policy with `instance_min_count: 0` (v0.1 schema)
  binds; the eventgenerator fires the scale-in trigger; the scaling engine parks the app (calls the
  activator, which binds the route service in the app's space) and scales the web process to 0.
  All confirmed in logs.
- **But the wake request returns HTTP 404** — `Requested route does not exist`. When a CF app scales
  to **0 running instances, route-emitter deregisters the route from Gorouter entirely** (404, not
  503). A bound route service only receives traffic **while the route remains mapped**; it does
  **not** keep an otherwise-unmapped route alive. So there is nothing for the route service to
  intercept once instances hit 0. **The route-service approach cannot implement scale-from-zero on
  standard CF.**

To keep a zero-instance route routable, the activator must be a **live backend** of that route. Every
way to achieve that was evaluated and is blocked on this foundation:

| Mechanism | Keeps route alive at 0? | mTLS to activator? | Blocker |
|---|---|---|---|
| Route-service binding (built) | ❌ (route deregisters → 404) | n/a | fundamental — doesn't keep route alive |
| Add activator app as route **destination** | ✅ | ✅ (identity-verified) | needs `route_sharing` flag — **disabled** here (cross-space) |
| Route-service UPSI shared cross-space | ✅ | ✅ | needs `service_instance_sharing` flag — **disabled** here |
| **routing-api** HTTP `ip:port` registration | ✅ | ❌ **plaintext only** (no `tls_port`/SAN in HTTP route model) | mTLS is mandatory → rejected |
| **NATS** `router.register` direct (route-registrar style) | ✅ | ✅ (schema has `tls_port`+`server_cert_domain_san`) | NATS unreachable from app (ASG); and a CF app can only self-register its **overlay** container IP, which **Gorouter cannot reach** (Gorouter is on the BOSH net, not the silk overlay). A CF app is not a BOSH VM with a Gorouter-routable address. |

**Conclusion.** The only mTLS-preserving, space-correct mechanism is **CF route destinations**
(share the parked route into the activator's space, add the activator app as a destination — it
becomes a healthy backend that keeps the route mapped and receives traffic over identity-verified
mTLS; on wake, remove the destination). This is gated solely on the **`route_sharing` feature flag**,
which is currently **disabled** on the target foundation. Enabling it is a foundation-operator
decision. Direct-to-Gorouter registration (routing-api / NATS) is not viable for a CF-app activator:
either plaintext-only or physically unreachable over the overlay.

The activator implementation below (route-service binding) is retained as the scaffolding but its
interception mechanism must be replaced with route-destination mapping once `route_sharing` is
enabled.

### 2.1.1 Update (2026-09-03): NATS backend registration works — for both keep-alive and readiness

The pessimistic conclusion above (line "Direct-to-Gorouter registration … not viable") was **wrong
about NATS** and is **superseded**. The overlay-IP objection does not apply: the activator does **not**
register its silk overlay IP. It registers the tuple its own cell's route-emitter already advertises
for the activator's route — `host = CF_INSTANCE_IP` (the Diego **cell** IP, on the BOSH net and
Gorouter-reachable), `tls_port = external_tls_proxy` from `CF_INSTANCE_PORTS`, `server_cert_domain_san
= CF_INSTANCE_GUID`. Publishing `router.register` with `uris=[parkedapp.domain]` and that tuple makes
Gorouter route the parked hostname to the activator **over mTLS with route integrity** (the backend
cert SAN matches the activator's real instance GUID). Gorouter does no ownership validation on
`router.register`. This is **live-validated**: a parked app's route resolves to the activator; the
wake request reaches it over mTLS. NATS reachability needs only an operator-settable ASG (done).

So the chosen mechanism is **NATS `router.register` (backend-only)**, not route destinations. The
route-service binding scaffolding is fully removed; §3–§5 below describe route-service mechanics and
are **historical** — read §3.1 / §5.x "NATS" notes for the shipped design.

**Readiness signal — routing-api was tried and abandoned.** The first cut used the CF routing-api
event stream (`GET /routing/v1/events`) as the "app is up" signal (Loop B). Live CI showed the
subscription returns **`401 Unauthorized`** from the event source even with a valid `routing_api_client`
token carrying `routing.routes.read` — so no `Upsert` ever arrived, the wake handler timed out at the
120 s readiness cap (`app-not-ready-in-time`), the held request was never forwarded, and the client
`curl` hung. (This was the actual CI failure — **not** the cooldown gate; the `BypassCooldown` fix
worked: the wake `POST /scale` returned 200 with no cooldown rejection.)

**Resolution — readiness over the same NATS bus.** route-emitter publishes `router.register` for a
real app instance's backend on the **same** bus the activator is already connected to (confirmed from
gorouter source `mbus/subscriber.go`: Gorouter subscribes to `router.*`; there is one route bus; since
the activator's own `router.register` reaches Gorouter on it, route-emitter's do too). The readiness
watcher now **subscribes to `router.register`** on the registrar's existing NATS connection and treats
"a registration for a parked URI from a backend that is **not** the activator (filtered by
`private_instance_id` / `server_cert_domain_san`, or host+`tls_port`)" as the readiness edge. This
removes the routing-api HTTP stream, the UAA `routing_api_client` dependency + credhub secret, and the
second transport entirely — one NATS connection now serves both keep-alive and readiness. Trade-off:
readiness latency is bounded by route-emitter's register cadence (worst case ~one emit cycle), and the
subscriber must self-filter its own keep-alive registrations. On the readiness Upsert the watcher
**deregisters the activator's own backend before releasing the held request**, so the forward to
`https://<host>` reaches the real app and cannot load-balance back to the activator during the window
both backends are pooled.

## 3. Interception mechanism: bind only while parked

The route service is a **user-provided service instance** (UPSI) whose
`route_service_url` is the activator's own public route. A route binding can only reference a
service instance **in the same space as the route**, and cross-space service *sharing* may be
disabled on the foundation (`service_instance_sharing` feature flag — observed disabled on
cf-deployment). Therefore the activator ensures a **per-space UPSI on demand**: when it parks an
app, it resolves the app's space and finds-or-creates a UPSI named `autoscaler-activator-rs` in
that space (pointing at the activator's route), then binds the app's routes to it. No global,
deploy-time UPSI and no cross-space sharing are needed.

```
# conceptually, per app-space, done by the activator at park time:
cf create-user-provided-service autoscaler-activator-rs -r https://<activator-route>   # if absent in that space
```

Bindings churn via the CF v3 API on the **route** (not the app):

- Bind:   `POST /v3/service_route_bindings` (route guid + UPSI guid). Synchronous for a UPSI.
- Unbind: `DELETE /v3/service_route_bindings/:guid`. Synchronous for a UPSI.
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
  The client appends `/routing/v1/events` itself, so the configured `routing_api.url` must be the
  **base** (`https://api.<domain>`), NOT `.../routing/v1` (that doubles the path → 404).
- **Reachability (verified on cf-deployment):** `https://api.<domain>/routing/v1/events` returns 200
  with a `routing_api_client` token; the routing-api is a public IP, so app egress is already
  covered by the platform's `public_networks` ASG — no custom security group is needed. (If a
  foundation restricts egress, add an ASG as we do for log-cache — `metricsforwarder/security-group.json`.)
- **Credentials:** UAA client `routing_api_client` (has `routing.routes.read`), secret pulled from
  **credhub at build time** (`scripts/build-extension-file.sh` via `credhub interpolate`,
  `/bosh-autoscaler/cf/uaa_clients_routing_api_client_secret`) → `activator-config` UPSI → read at
  runtime as VCAP creds. Mirrors the eventgenerator log-cache UAA plumbing
  (`scripts/extension-file.tpl.yaml`, `models.UAACreds`).

### 7.2 Who issues the bind: engine vs activator

Two implementation options for §5.1 step 3:
- **Engine binds** synchronously before scaling to 0 (keeps ordering trivially correct; engine
  gains CF route-binding capability).
- **Engine notifies activator** to park, activator binds and confirms, then engine scales
  (keeps route logic in one place; adds a handshake).
PoC can start with whichever is simpler; the ordering contract (bind → confirm → scale-0) is the
invariant.

### 7.3 Control API vs route-service on one route (PoC-only)

The activator's engine-facing park/unpark control API and the user-facing route-service catch-all
are **co-hosted on the activator's single public route** (its CF server) in the PoC. A CF-app
activator is only reachable via its public route through Gorouter, so the old mTLS server (a
BOSH-era artifact where components had stable IPs) is vestigial and now a 404 stub. The two surfaces
need opposite auth — the control API is XFCC-authed (only the scaling engine may park), the
route-service must not be (it carries arbitrary end-user traffic) — which is achieved with a
gorilla/mux subrouter: XFCC middleware is applied only to the control subrouter, registered before
the unauthenticated catch-all so its specific `/v1/apps/{appid}/park` paths match first.

**Revise for production:** split the control API onto a dedicated `cf-` route (XFCC) separate from
the public route-service route, mirroring the scalingengine `HOST` vs `CF_HOST` split, to remove any
path collision between `/v1/apps/*/park` and a real app's own routes.

## 8. Rejected alternative: direct `routing-api` ip:port registration

Register the activator's `ip:port` as the parked route's backend via
`POST /routing/v1/routes` (TTL-refreshed), unregister on wake. This was the 2019 BOSH design and is
rejected now only because a CF-app activator has no stable, Gorouter-reachable IP (§2.2). Kept here
because if the activator were ever run on a stable-IP substrate again, it becomes viable — and it
avoids the always-possible route-service latency hop.

## 9. Open questions / risks to resolve during PoC

1. **Wake timeout — how long to hold a request while the app cold-starts (unresolved design
   decision).** The activator holds the incoming request until the app is ready, bounded by
   `ReadinessTimeout` (currently 120s). There is real tension here:
   - **Too short** → we return 503/Retry-After before an app that legitimately takes a while to
     start has come up, defeating the "client only notices a slower request" goal. The right upper
     bound should arguably **respect the app's / platform's configured startup + readiness health
     check timeouts** (CF `health-check-invocation-timeout`, `timeout`, and the platform default),
     since those already express "how long this app is allowed to take to become healthy". Reading
     those per-app and sizing the wait accordingly (rather than a fixed activator constant) is the
     principled approach.
   - **Too long** → we hold client connections (and the activator's own resources) for the full
     window; the client/Gorouter may time out anyway, and a crash-looping app would tie up held
     requests. Need an upper cap regardless of the app's configured timeout.
   Decide: fixed cap vs. derived-from-app-config vs. min(app-config, hard-cap); and what the held
   request sees on timeout (503 + Retry-After today). **Not yet designed — flagged for follow-up.**
2. **Cooldown vs wake — RESOLVED.** The scale-to-zero set a cooldown that suppressed the immediate
   wake scale-up (observed live: "scaling ignored: App in cooldown"). Fixed via
   `Trigger.BypassCooldown`, which the activator's wake sets so the engine scales up despite the
   cooldown. Keep in mind for re-park stability (item 8).
3. **Activator HA** — multiple activator instances register as backends for a parked route (Gorouter
   load-balances across them); the held request lives on the instance Gorouter chose. Parked-app
   registry must be reconstructable via reconcile (§5.4) rather than shared in-memory state, and each
   instance must publish its own tuple.
4. **NATS registration correctness on the target** — validated live that Gorouter accepts the
   cross-app `router.register` and routes to the activator over mTLS. Confirm behaviour holds across
   Gorouter/route-emitter restarts and the route-emitter's own (real-app) registrations after wake
   (last-writer / load-balance handoff — see §3).
5. **Multiple / path-based routes per app** — register all of an app's route URIs; handle path routes.
6. **Non-HTTP / no-route apps** — PoC scopes to HTTP routes. Apps with no routes: scale-to-zero is
   fine but only a schedule can wake them (no request trigger).
7. **Security** — the activator holds the NATS route-registration cert = ability to register routes
   for ANY app on the foundation (Gorouter trusts NATS unconditionally). Real trust escalation for
   operator/security review. Also lock down the engine↔activator mTLS and the ASG egress. (Deferred
   to a GitHub issue.)
8. **Re-park loop stability** — avoid flapping between park and wake for bursty-but-idle apps
   (respect cooldown / a minimum-parked dwell time). Note this interacts with the BypassCooldown of
   item 2 — the wake bypasses cooldown, but the subsequent idle-driven re-park should not thrash.

## 10. Out of scope for PoC

Metrics/observability polish, activator self-autoscaling, TCP routes, request-buffering
backpressure limits, and production hardening of the bind/scale atomicity beyond the documented
ordering.
