---
name: verify
description: Build, run, and drive the SalehCard Go API locally to verify changes end-to-end.
---

# Verify the SalehCard API

## Launch

```bash
cd api
# The local Mongo (:27017) is a replica-set member that advertises itself as
# "mongo:27017" (Docker hostname) — without directConnection=true the driver
# discovers that unreachable name and times out.
export MONGO_URI="mongodb://localhost:27017/?directConnection=true" \
       ENV=development JWT_SECRET="verify-secret-123" PORT=8099
go run ./cmd/seed        # idempotent; review-seed may dup-key on a dirty dev DB (harmless, but it aborts later seeds)
go run ./cmd/server      # run in background; ready when GET /api/v1/products → 200
```

Pick a PORT not used by other sessions (8080/8090 are often taken).

## Seeded identities (password `password123`, use `X-Client: mobile` to get tokens in-body)

- `admin@salehcard.local` — admin (super admin, all perms)
- `customer@salehcard.local` — customer
- `gamehub.store@salehcard.local` — reseller, Gold tier (12%)
- `topup.pro@salehcard.local` — reseller, Silver (8%)

```bash
curl -s -X POST :8099/api/v1/auth/login -H "Content-Type: application/json" \
  -H "X-Client: mobile" -d '{"email":"...","password":"password123"}'
# → data.accessToken
```

## Gotchas that cost time

- **KYC gates every purchase**: a fresh user 403s `KYC_REQUIRED` on POST /orders.
  Unblock via the real flow: `POST /api/v1/kyc` (required fields: fullName,
  dateOfBirth YYYY-MM-DD, placeOfBirth, placeOfResidence, documentType,
  documentNumber, documentFrontUrl — any `http(s)://…/kyc/….jpg` URL passes
  validation), then approve as admin: `PUT /api/admin/kyc/{submissionId}`
  `{"status":"approved"}`.
- **Order payload**: `{"items":[{"productId","variantId","qty":1}],"currency":"USD","paymentMethod":"wallet"}`
  with an `Idempotency-Key` header. The quantity field is `qty`, not `quantity`.
- **Code products need stock**: `POST /api/admin/products/{id}/codes` with
  `{"codes":[{"code":"..."}]}` (objects, not strings).
- Windows Python defaults to cp1252 — pass `encoding='utf-8'` when reading
  captured JSON responses.
