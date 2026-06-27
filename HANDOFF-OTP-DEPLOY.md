# Handoff — Deploy phone-OTP + Monty SMS to production

For the **deployment / CI-CD session**. Companion to `DEPLOYMENT.md` (resources,
proxy workaround, redeploy recipes) and `DEPLOY-CREDS.local.md` (secrets, gitignored).
This doc is what's needed to ship the new **phone-OTP authentication + Monty SMS**
feature to Azure.

## What's new (and why a deploy is needed)

The API gained **phone-OTP auth** and an **SMS provider layer**; both are live and
verified **locally** but **not in prod**. The deployed `salehcard-api` still runs the
old image without these routes, so the mobile app can't do OTP against prod yet.

- New endpoints (all under the already-CORS-allowed `/api/v1/auth/*`, JSON bodies —
  **no CORS change needed**):
  - `POST /auth/otp/request` `{phone}` — sends an OTP SMS
  - `POST /auth/otp/verify` `{phone, code, password?}` — verifies, finds-or-creates the
    account, returns tokens (same `AuthResponse` shape as login; mobile gets the refresh
    token in-body via `X-Client: mobile`)
  - `POST /auth/login-phone` `{phone, password}`
- SMS is a hexagonal port/adapters layer (`api/internal/platform/sms`): `Sender` port +
  `monty.go` / `twilio.go` / `log.go`, chosen by `SMS_PROVIDER`. **Monty** (Lebanon) is
  the active provider; verified end-to-end with a real SMS (sender `Arya`).

**Code state:** branch **`feat/flutter-mobile-skeleton`** (HEAD `faeb51e`), pushed to
`github.com/AliSleiman0/SalehCard`. ⚠️ `DEPLOYMENT.md` notes prod last shipped from
`feat/catalog-migration-browse` (PR #3) — **reconcile branches**: the pipeline must build
the commit that contains the OTP code (this branch), or merge it to whatever `main`/release
branch CI deploys from.

## Deploy checklist

### 1. Build & deploy the API image
Same recipe as `DEPLOYMENT.md → Redeploy recipes → API` (cross-compile Linux binary →
`Dockerfile.prebuilt` → `docker push` to `salehcardprodacr` → `az rest` PATCH
`config/web` `linuxFxVersion` to the new tag → restart). **CI-side build (GitHub Actions)
is preferred** — it compiles the Go binary cloud-side and sidesteps the local
corporate-proxy TLS issue entirely (this is open-item #4 in `DEPLOYMENT.md`; needs a
`workflow`-scope token).

### 2. Add App Service app settings (`salehcard-api`)
New settings (values/secret in `DEPLOY-CREDS.local.md → Monty Mobile`):

| Setting | Value | Notes |
|---|---|---|
| `SMS_PROVIDER` | `monty` | selects the active adapter |
| `MONTY_USERNAME` | `ozconSer` | |
| `MONTY_API_ID` | `hzed4BCe` | the `apiId` query param |
| `MONTY_ACCESS_TOKEN` | *(secret)* | `X-Access-Token`; from `DEPLOY-CREDS.local.md` |
| `MONTY_SENDER_ID` | `Arya` | only approved sender so far ("SalehCard" not yet registered with Monty) |
| `MONTY_BASE_URL` | *(optional)* | defaults to `https://sms.montymobile.com` |
| `MONTY_CAMPAIGN` | *(optional)* | `campaignname`; defaults to username |

Optional OTP tuning (all have safe defaults — only set to override):
`DEFAULT_COUNTRY_CODE=+961`, `OTP_LENGTH=6`, `OTP_TTL=5m`, `OTP_RESEND_INTERVAL=60s`,
`OTP_MAX_ATTEMPTS=5`. (App settings are set via `az rest` PATCH `config/appsettings`, since
`az webapp` fails through the proxy — see `DEPLOYMENT.md`.) Store `MONTY_ACCESS_TOKEN` as a
real secret (GitHub Actions secret / Key Vault), never in git.

### 3. Whitelist the Azure outbound IPs on Monty  ← easy to miss
Monty rejects sends from non-whitelisted IPs as **`-8 Invalid Source`** (looks like a
sender problem but is IP-gated). Only the **dev** egress `213.204.66.98` is whitelisted.
Get the App Service outbound IPs and have Monty (account `ozconSer`) whitelist them:
```
az webapp show -g salehcard-prod -n salehcard-api --query outboundIpAddresses -o tsv
az webapp show -g salehcard-prod -n salehcard-api --query possibleOutboundIpAddresses -o tsv
```
Whitelist the **full `possibleOutboundIpAddresses` set** (App Service rotates within the
pool), or front the app with a **NAT Gateway** for one stable egress IP. (This is the same
IP set as `DEPLOYMENT.md` open-item #5 — Atlas network-access tightening.)

### 4. Database — no manual migration
`EnsureIndexes` runs at API startup against Atlas (`salehcard` db). On first boot of the new
image it will, idempotently:
- add a **sparse-unique** index on `users.phone`;
- create collection **`otp_codes`** with a unique index on `phone` + a **TTL** index on
  `expiresAt` (auto-expires spent codes);
- **reconcile the `users` email index** to sparse-unique — it drops a pre-existing
  non-sparse `email_1` and recreates it sparse (so phone-only accounts don't collide on a
  null email). Brief and safe on Atlas M0 (which supports sparse/partial/TTL). Just be aware
  it runs once at startup.

### 5. Mobile app (separate from API deploy)
The **currently installed release APK predates the OTP UI** (old demo-bridge login) **and**
points at Azure. After the API is live with OTP, **rebuild the release APK** from this branch
(it now has the code/password OTP screens) targeting `https://salehcard-api.azurewebsites.net/api/v1`
and reinstall. No app code change needed beyond a rebuild.

## Verify in prod
```bash
curl https://salehcard-api.azurewebsites.net/health        # {"status":"ok"}
# Real OTP (sends an SMS to the number via Monty/Arya):
curl -X POST https://salehcard-api.azurewebsites.net/api/v1/auth/otp/request \
  -H 'Content-Type: application/json' -d '{"phone":"70081637"}'   # {"success":true,"data":{"sent":true}}
# App Service log should show:  INFO sms sent via monty to=+961... id=...
# Then verify with the received code:
curl -X POST https://salehcard-api.azurewebsites.net/api/v1/auth/otp/verify \
  -H 'Content-Type: application/json' -d '{"phone":"70081637","code":"<6-digits>"}'
```
If `otp/request` returns 500 / the log shows `monty error -8 Invalid Source`, the Azure
outbound IP isn't whitelisted on Monty yet (step 3) — that's the most likely prod failure.

## Follow-ups
- Register **"SalehCard"** as a Monty sender ID, then flip `MONTY_SENDER_ID` from `Arya`.
- A Monty send failure currently surfaces as a generic `500 INTERNAL_ERROR` in
  `RequestOTP` — consider mapping to a clearer code (e.g. 502).
- Rotate `MONTY_ACCESS_TOKEN` (pasted in chat during testing).
