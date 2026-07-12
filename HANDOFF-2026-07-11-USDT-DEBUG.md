# HANDOFF — 2026-07-11: USDT "payments not received" — DEBUG SESSION

> Written by the 07-11 chain-scan session; **reviewed and amended by the 07-10
> build/ship session** (the one that shipped PR #69/#71, enabled BEP20 in prod
> and built the picker APK). Amendments are marked; the APK paragraph and
> root-cause #2 replace earlier incorrect/missing content.

## Symptom (as reported by Ali, 2026-07-11)

Payments are not being received. No further detail captured (which network,
who paid, txHash, amount — all unknown). Next session = debugging.

## ⚡ KEY FINDING already established (checked 2026-07-11, this session)

**The money never arrived on-chain at our addresses.** Checked both chains
directly, independently of our system:

- **TRC20** `TLRaHegyg2grMQqX85nJyCzbdRtvM5nCDn`: public TronGrid query
  (`/v1/accounts/<addr>/transactions/trc20?only_to=true`, all tokens) — the
  address is an ACTIVE business wallet (regular 100–2000 USDT inbound
  transfers), but the **latest inbound transfer is 2026-07-06 05:43 UTC** —
  four days BEFORE go-live. Zero transfers since.
- **BEP20** `0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc`: eth_getLogs sweep via
  rpc-bsc.48.club over the last ~34,000 blocks (≈28h), first filtered to USDT,
  then **unfiltered (ANY ERC20-style token)** — **zero inbound transfers**.

So this is almost certainly NOT a watcher/matching bug: there was nothing on
chain to match. The failure is upstream — the payer's transaction was never
broadcast, went to a different address, or a different network.

Server health at last observed boot (2026-07-10 18:50 UTC log):
- Both enable lines present (`trongrid network=trc20` + `jsonrpc network=bep20`),
  watcher started, no `transfer list failed` / `chain lookup failed` errors.
- App settings verified again 2026-07-11: `USDT_PROVIDER=trongrid`,
  `USDT_ADDRESS=TLRa…` (exact), `USDT_BEP20_PROVIDER=jsonrpc`,
  `USDT_BEP20_ADDRESS=0x5e0a…`, `TRONGRID_API_KEY` present
  (`ETHERSCAN_API_KEY` present but unused). Always On = true.
- ⚠️ Downloaded container logs only reach 18:50 UTC 07-10 — during debugging use
  **live** `az webapp log tail -n salehcard-api -g salehcard-prod` instead.

## Debug checklist (in order)

1. **Get the facts from the payer** (Ali or the client — whoever "paid"):
   - Screenshot of the withdrawal/send screen + the **txHash** if any.
   - Which network they selected, which address they pasted, exact amount.
   - **Was the withdrawal zero-fee/instant or marked "internal transfer"?**
     If yes → root cause #2 below (off-chain exchange-internal transfer; the
     money likely IS in the client's exchange account — have him check his
     funding/deposit history). An "internal transfer" ID is NOT an on-chain
     txHash and will not resolve on tronscan/bscscan.
   - If there is NO txHash → the transfer never executed. Common causes:
     Binance withdrawal stuck on verification/2FA hold, cancelled, insufficient
     balance for the ~1 USDT TRC20 fee, or the payer only *created the intent
     in the app* and believed that was the payment (UX misread — plausible
     given this client's skill level; see the WhatsApp saga in
     HANDOFF-2026-07-10.md).
2. **If a txHash exists**: look it up on tronscan.org / bscscan.com → see where
   the funds actually went (wrong address? wrong network? their own wallet?).
3. **Check what intents exist server-side**: admin console → Payments
   (status filters; "Unmatched deposits" tab — expected EMPTY per the chain
   scans). For DB-level detail use the prod-DB direct access recipe
   (memory: salehcard-prod-db-direct-access — temp firewall rule for the
   213.204.66.x egress IP + MONGO_URI from DEPLOY-CREDS.local.md, delete rule
   after; `api/cmd/dbtool` for guarded reads). Look at `payment_intents`:
   status/network/amountExpectedMicros/expiresAt of the attempts. Telling
   detail: if ZERO `network:"bep20"` intents exist at all, the BEP20 path was
   never even attempted from any app (only Ali's phone has the picker build,
   see below) — the incident is then entirely about TRC20 intents.
4. **Only if a real tx DID land at our address** (contradicting today's scan):
   then it's our bug — live-tail logs during a watcher tick and check, in
   order: exact-amount match (payer rounded → should appear as unmatched
   deposit), BlockTime≥CreatedAt guard, known-txHash filter, TRC20 TronGrid
   min_timestamp, BEP20 jsonrpc cursor/lookback/min-confirmations (15 blocks
   ≈ 45s lag is normal; free-node quirks documented in the memory file
   salehcard-usdt-onchain-payments §v3 and PR #71).
5. **Re-check settings drift**: `USDT_BEP20_ADDRESS` mysteriously vanished once
   on 07-10 17:47 UTC (restored). If symptoms point at BEP20 being dark,
   re-list the app settings first — takes 10 seconds.

## Likely root causes, ranked

1. **The transfer was never sent** (intent created in-app ≠ payment; or
   Binance withdrawal never completed). Fits ALL evidence.
2. **⚠️ Exchange INTERNAL transfer — payment SUCCEEDED but OFF-CHAIN**
   (added by the 07-10 build session; fits ALL evidence equally): the client
   is a custodial-Binance user and both shared addresses are very plausibly
   **exchange deposit addresses** (TLRa… receives regular 100–2000 USDT
   business inflows — classic deposit-address pattern). When a payer
   withdraws from Binance TO another Binance deposit address, Binance
   executes an **internal transfer: zero fee, instant, NO on-chain
   transaction, no real txHash** — the client gets the money in his exchange
   account while TronGrid/eth_getLogs (and therefore our watcher) see
   NOTHING, forever. Verify: ask the payer if the withdrawal showed
   zero-fee/instant/"internal transfer", and ask the CLIENT to check his
   exchange funding history for the amount. If confirmed, this is a
   **design-level gap of shared-address mode with a custodial deposit
   address**, not a bug: same-exchange payments can never auto-confirm.
   Mitigations to discuss: admin manually credits the wallet after the client
   confirms receipt (plain wallet adjustment — the deposit-attribute flow
   needs a txHash, which doesn't exist here); and/or in-app copy telling
   Binance payers to expect manual confirmation; and/or long-term move to a
   non-custodial wallet address (or the xpub upgrade path).
3. Sent to the wrong destination (payer's own Binance deposit address, an old
   address from the WhatsApp thread — the client circulated several candidate
   addresses there, including his own deposit addresses).
4. (Distant) our watcher missed a real on-chain transfer — contradicted by the
   direct chain scans; only revisit if a txHash surfaces pointing at our
   address.

## Chain-checking snippets (reusable)

- TRC20 inbound (all tokens, no key):
  `curl "https://api.trongrid.io/v1/accounts/TLRaHegyg2grMQqX85nJyCzbdRtvM5nCDn/transactions/trc20?only_to=true&only_confirmed=true&limit=10&order_by=block_timestamp,desc"`
- BEP20 inbound: eth_getLogs on `https://rpc-bsc.48.club` (≤4999-block chunks),
  topics `[Transfer, null, 0x000…<addr-no-0x>]`; omit `address` to catch
  any token. (48.club prunes old blocks; ~1M lookback max.)

## System state / context

- Prod: TRC20 (PR #68) + BEP20 (PR #69/#71, jsonrpc free-node provider) both
  ENABLED since 2026-07-10 18:50 UTC. Full architecture + BSC free-node
  gotchas: memory `salehcard-usdt-onchain-payments`, plans
  `plan-full-implementation-cuddly-hummingbird.md` +
  `plan-impelmentation-of-bep20-tingly-clock.md`.
- **CORRECTION (from the 07-10 build session): the release APK WITH the BEP20
  picker WAS built and installed** on Ali's phone (R3CT90MEVFM) at ~21:56
  local on 07-10 — built from `origin/main` @ `35c2537` (PR #71) with the prod
  `--dart-define`, `adb install -r` → Success. Copies for forwarding:
  phone `Download/SalehCard-2026-07-10.apk` + PC
  `C:\Users\user\Downloads\SalehCard-2026-07-10.apk`. So a BEP20 attempt from
  ALI's phone was possible. **Unknown: whether the client received/installed
  it** — the plan was WhatsApp; note WhatsApp sometimes blocks/mangles `.apk`
  attachments (workaround: rename to `.apk1` or share via Drive link). If the
  payer is the CLIENT on an older build, his app omits `network` → trc20 only.
- Local git: repo on branch `feat/usdt-bep20-second-network`. The uncommitted
  modifications are **redundant working-tree copies of already-merged work**
  (PR #69/#70/#71 shipped the same content from worktrees) — EXCEPT the
  genuinely-unshipped leftovers: `bridge/**` (Alfa USSD debug),
  `api/cmd/inputlabels/`, HANDOFF files, stray `.exe`s. `git status` before
  switching branches, don't clobber. `main` is at PR #71 (`35c2537`).
- Watcher behavior worth knowing while debugging: the BEP20 (and TRC20 shared)
  chain scan runs ONLY while at least one open/in-grace intent exists on that
  network — an idle-network tick makes zero chain calls and logs nothing, so
  log silence ≠ failure. The BEP20 jsonrpc cursor is in-memory: an app restart
  re-scans the whole open window (bounded, dedup makes it harmless).
