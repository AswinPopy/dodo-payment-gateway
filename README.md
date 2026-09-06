# Dodo Payment Gateway

Invoice API on `:8080`. Mock card processor (PSP) on `:8081`. Amounts are integer minor units (cents). Full request and response shapes: [docs/openapi.yaml](docs/openapi.yaml). API shapes live in [docs/openapi.yaml](docs/openapi.yaml).

```
Invoice API  :8080  ──HTTP──►  Mock PSP  :8081
     │
     ▼
 PostgreSQL
```

## Run

**Docker (all three processes):**

```bash
docker compose up --build
```

- API: `http://localhost:8080`
- Mock PSP: `http://localhost:8081`

**Local (two terminals + Postgres):**

Postgres should be reachable as `DATABASE_URL`. If unset, the server uses:

`postgres://postgres:postgres@localhost:5433/dodo_psp?sslmode=disable`

```bash
# terminal 1
go run ./cmd/mock-psp

# terminal 2
go run ./cmd/server
```

Migrations run when the server starts.

## Curl examples

Use a unique `Idempotency-Key` on every **new** payment. Reusing a key on a different invoice returns `409`.

### 0. Business + API key (once)

Customer, invoice, and pay all need a Bearer token.

```bash
export BASE=http://localhost:8080

export BID=$(curl -sS -X POST $BASE/businesses \
  -H 'Content-Type: application/json' \
  -d '{"name":"Acme"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

export KEY=$(curl -sS -X POST $BASE/api-keys \
  -H 'Content-Type: application/json' \
  -d "{\"business_id\":\"$BID\"}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['api_key'])")
```

### 1. Create customer

```bash
export CID=$(curl -sS -X POST $BASE/customers \
  -H "Authorization: Bearer $KEY" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

echo "$CID"
```

### 2. Create invoice

`amount` is cents. Invoice starts as `DRAFT`.

```bash
export INV=$(curl -sS -X POST $BASE/invoices \
  -H "Authorization: Bearer $KEY" \
  -H 'Content-Type: application/json' \
  -d "{\"customer_id\":\"$CID\",\"currency\":\"USD\",\"amount\":1000}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

curl -sS -w '\n' $BASE/invoices/$INV -H "Authorization: Bearer $KEY"
```

### 3. Pay — success (`tok_success`)

```bash
curl -sS -w '\n' -X POST $BASE/invoices/$INV/pay \
  -H "Authorization: Bearer $KEY" \
  -H "Idempotency-Key: pay-ok-1" \
  -H 'Content-Type: application/json' \
  -d '{"card_token":"tok_success"}'
```

Expect attempt `SUCCEEDED` and a `psp_transaction_id`. `GET /invoices/$INV` is `PAID`.

### 4. Pay — failure (`tok_card_declined`)

Use a **new** invoice. A paid invoice cannot be charged again.

```bash
export INV_FAIL=$(curl -sS -X POST $BASE/invoices \
  -H "Authorization: Bearer $KEY" \
  -H 'Content-Type: application/json' \
  -d "{\"customer_id\":\"$CID\",\"currency\":\"USD\",\"amount\":2500}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

curl -sS -w '\n' -X POST $BASE/invoices/$INV_FAIL/pay \
  -H "Authorization: Bearer $KEY" \
  -H "Idempotency-Key: pay-fail-1" \
  -H 'Content-Type: application/json' \
  -d '{"card_token":"tok_card_declined"}'
```

Expect attempt `FAILED` and `failure_code: card_declined`. Invoice stays `OPEN` (not paid). `tok_insufficient_funds` behaves the same with a different code.

Other tokens the mock PSP understands: `tok_timeout` (returns `PENDING` in ~2s), `tok_network_error` (also `PENDING`).


Video link : https://drive.google.com/file/d/18rKHJvXId-SF7SPzkSMUrtE95VDVxg3R/view?usp=sharing
