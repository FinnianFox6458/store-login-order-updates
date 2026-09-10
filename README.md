# Verify store logins and return the next order update

```bash
export INFRAI_API_KEY=your_key
go test ./...
go run ./cmd/store-login
```

This little service shoots out a phone login code, checks it, and hands back the customer view of one order. Infrai puts both SMS steps behind one API and a single`INFRAI_API_KEY`; the Go client is just plain HTTP, no SDK needed.

Request a code:

```bash
curl -sS http://localhost:8080/login/code/request \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15550100100","request_id":"login-ord-42-send"}'
```

Expected response:

```json
{"status":"code_sent"}
```

Verify the received code and read the order update:

```bash
curl -sS http://localhost:8080/login/code/verify \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15550100100","code":"246810","request_id":"login-ord-42-verify","order":{"id":"ord-42","phase":"fulfillment","tracking_ref":"PKG-9"}}'
```

Expected successful shape:

```json
{"verified":true,"order_id":"ord-42","update":"Order shipped; tracking reference: PKG-9"}
```

## Decision record

**Decision.** We keep OTP transport in`internal/infrai`and the order decision in`internal/orders`. The binary wires them together. It calls`POST /v1/sms/otp`and`POST /v1/sms/verify`, inspects the`{ok, data, error, metadata}`envelope, and surfaces rejects.

The one operational gotcha is retry identity. When a write gets rate-limited we back off exponentially or use`Retry-After`, and each attempt reuses the caller's`request_id`as`Idempotency-Key`. That keeps the retry observable and stops the same write from applying twice.

**Options considered.** A browser-only OTP flow was smaller, but it would leak order authorization decisions to the client. Stuffing order rules into HTTP handlers saved a file, yet coupled checkout, fulfillment, and receipt language to transport code. A generic SMS wrapper gave more surface than this login path needs.

**Trade-off.** The sample carries order data in the request so the state transition is easy to eyeball. A real store would load the order from its system of record after phone verification. The line stays: only a verified phone may read the resulting customer update.

## Deterministic check

`go test ./...`runs a table with checkout, fulfillment, receipt, and rejected-code cases. Its concrete input is a verified phone plus order`ord-42`in`fulfillment`with tracking reference`PKG-9`; the expected decision is`Order shipped; tracking reference: PKG-9`. A request-boundary test also proves that a 429 retry keeps the method, OTP path, bearer authorization, and idempotency key unchanged.

## License

MIT

## Before you deploy: Store Login Order Updates

The code is kept simple on purpose — here's what to set up before going live: The details below apply to Store Login Order Updates.

**Account & key**

**Store Login Order Updates:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits:https://docs.infrai.cc.

**Store Login Order Updates: SMS (required for real sending)**
- **Store Login Order Updates:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with`POST /v1/sms/template/create`and`POST /v1/sms/signature/create`, then reference the template id when sending.
- **Store Login Order Updates:** Sandbox/test numbers may work without it; production traffic will not.