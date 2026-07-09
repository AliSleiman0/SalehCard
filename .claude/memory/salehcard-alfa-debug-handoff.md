---
name: salehcard-alfa-debug-handoff
description: Alfa bridge debug handoff — RESOLVED the "Alfa balance always 0" bug (device-side balance-reply parser); HANDOFF-ALFA-DEBUG.md still maps the wider flow
metadata: 
  node_type: memory
  type: project
  originSessionId: 949cfd1f-2fb2-4dc0-b66b-6a091e98e117
---

**RESOLVED 2026-07-08 — the reported symptom was "Alfa balance always shows 0"**
(Touch/MTC already fixed). Root cause was device-side, NOT the recharge/settle
path: `bridge/app/.../CommandExecutor.kt` `getBalanceFromReply` had ONE regex
hard-wired to Touch's `*220#` reply shape (`USD 5.00 Exp:10-06-27`, currency-first)
and did a positional `split(" ")`/`substring(4)`. Alfa's `*11#` reply is a
different shape — `1.01 USD till 04/09/2026` (amount-first, "till", slash date,
4-digit year) — so it never matched and balance stayed 0.00. Fix: rewrote the
parser to try both patterns via capture groups (amount, validity); added
`BalanceReplyParseTest` (5 cases, all green via `./gradlew :app:testDebugUnitTest`).
Blast radius was **display-only** — the Alfa transfer min-balance pre-check is
gated `isTouch` (requiredBalance=0 for Alfa), so a 0 balance never blocked Alfa
transfers/recharges. `AlfaBalanceUSSD=*11#` in config was already correct → **no
API/config change, no API redeploy**. BUT this is an Android APK change: the prod
bridge phone must be reinstalled with a fresh bridge APK (`./gradlew installDebug`)
for it to take effect. JAVA_HOME on this box: use Android Studio JBR
(`C:/Program Files/Android/Android Studio/jbr`) — the env's `C:\mip\jdk21` is broken.

**Also RESOLVED same session — "Touch transfer works but shows FAILED on Bridge
admin"** (failReason = "SMS sending failed on chunk 1/1", reply sender = shortcode
1199). Root cause was NOT reply-text: `CommandExecutor.sendSms` registered the SMS
"sent" PendingIntent receiver with `RECEIVER_NOT_EXPORTED`; on Android 13+ that
cross-process broadcast (delivered by the telephony system process) never fires,
so sendSms times out → returns false → the chunk aborts as "SMS sending failed"
BEFORE ever awaiting the operator's "…transferred…" reply, even though the radio
sent the SMS and Touch moved the money. Fix: (1) `RECEIVER_EXPORTED` (safe — action
has a random per-send UUID) — repairs ALL send paths (transfer, rechargeAlfa,
SEND_SMS); (2) made the sent-ack NON-fatal in the transfer chunk loop AND
rechargeAlfa — on a missing ack, log a warning and fall through to await the
operator reply as source of truth (financial state still only mutated on a
confirmed success reply, so no double-send). Compiles + balance test green via the
JBR. Note the device success-detection is hardcoded ("success"/"transferred"), NOT
the server's configurable BRIDGE_SUCCESS_PATTERNS — if a real reply lacks those
words, revisit. Alfa transfer_credit shares the SAME `transferCredits`+`sendSms` path (dispatcher
routes both touch & alfa TRANSFER_CREDIT there), so it's fixed by the same two
edits — no alfa-specific change. Alfa success reply "…successfully transferred…"
matches the hardcoded check. CONFIG CAVEAT: prod sends alfa transfer to shortcode
**1313**, but the shipped default `BRIDGE_ALFA_TRANSFER_DEST`=1399 (1313 is the
default *recharge* dest); confirm prod app-setting `BRIDGE_ALFA_TRANSFER_DEST=1313`
so the SMS dest + reply-sender allowlist (`["alfa", transferDest]`) line up.

**Third symptom — "MTC transfer says not enough" + admin Touch balance 0.00
(2026-07-08):** ROOT CAUSE = the OLD balance parser reads 0. Prod reserve is NOT
the blocker — confirmed via `az webapp config appsettings list` that prod already
sets `BRIDGE_TOUCH_MIN_BALANCE=0.5` (and ALFA=0.5, BRIDGE_ENABLED=true), so a $1
transfer needs only $1.66 and the real SIM balance $4.26 (raw *220# reply "USD 4.26
Exp:10-06-27;MI-44GB:… Send WX2 to 1100") would clear it. It fails because the
device's in-memory `touchSimBalance` is 0. Why: the old `getBalanceFromReply` did
`lineOne.split(" ")[1]` + `substring(4)` — positional, so ANY irregular whitespace/
newline in the USSD text (common) shifts the index → `toDoubleOrNull() ?: 0.00` →
balance 0, but validity slice is non-empty so checkBalance still returns SUCCESS
(5000) = a "successful" zero. That 0 heartbeats to admin AND fails the transfer
pre-check. My capture-group rewrite of getBalanceFromReply (the same Alfa fix) is
IMMUNE to whitespace and reads 4.26 — proven by new BalanceReplyParseTest cases
(real multi-line reply + double-space + newline). So NO reserve/env change needed;
the single fix for all three symptoms is getting the new APK onto the real bridge
phone. IMPORTANT: prod bridge phone is STILL ON OLD APK (Alfa checks still fail 5022
after my install) — the Samsung I adb-installed to is NOT the executing bridge device;
new APK must reach the real bridge phone (WhatsApp) + relaunch before anything is fixed.
Azure read recipe that worked: `az webapp config appsettings list --name salehcard-api
--resource-group salehcard-prod` (az logged in locally, reads OK in auto mode).

**THE ACTUAL ROOT CAUSE (found live via adb logcat on the Samsung, 2026-07-08):**
the device config decode was CRASHING. `ConfigurationDTO.deviceId` was typed `Int`
but the server sends the Mongo ObjectID hex string (`d.ID.Hex()`) → Gson threw
`NumberFormatException: For input string: "6a4c20ae…"` on the WHOLE payload →
`fillProviderStore` never ran → device silently used HARDCODED DEFAULTS, incl.
`touchMinimumAllowedBalance=20.0` (ignoring prod's 0.5) AND default templates/
patterns. THAT is why a $4.26 SIM was refused a $1 transfer regardless of the parser
— config (reserve 0.5) never reached the device. Fix: typed `deviceId` as String in
ConfigurationDTO + ProviderStore (only stored, never numeric) + ConfigurationDTO
ParseTest. Confirmed on-device: after the fix the "Config not loaded" warning is
gone. (Note: Gson does NOT apply Kotlin default values for JSON-omitted fields —
they come back 0/null, not the declared default; runtime guards poll/heartbeat with
.coerceAtLeast, and the server sends all fields anyway.) The Samsung SM-S908N is the
test bridge but has ONLY a Touch SIM ("Device doesn't have dual SIM (found 1)"/
"No alfa SIM detected") — Alfa needs its SIM physically in the phone.

Live-debug recipe that worked: adb at C:/Users/user/AppData/Local/Android/Sdk/
platform-tools/adb.exe; `am force-stop` + `am start -n com.example.mobilebridgev2/
.MainActivity` then `logcat -s BridgeReporter:V AndroidRuntime:E *:F` — startup auto-
runs a Touch CHECK_BALANCE. (Internal balance-check reportResult 404s are harmless —
local UUID the server doesn't know; balance still set locally + via heartbeat.)

The 3 CONFIRMED-GOOD fixes (balance parser, transfer sent-ack, config deviceId
decode) are MERGED TO MAIN (branch fix/bridge-balance-parse-and-transfer-ack →
merged; commits 25d8c5e, b6e04c9). They are Android-APK-side — the WhatsApp APK at
C:\Users\user\Downloads\salehcard-bridge-2026-07-08.apk has them; real bridge phone
needs it installed + app relaunched. (The deviceId config-decode fix is the critical
one — without it the device ignored ALL server config incl. the 0.5 reserve.)

**Alfa transfer FORMAT still OPEN (2026-07-08):** Alfa rejected the transfer SMS
with "Wrong format. The correct format is 03/70/71/76/79/81XXXXXXR<credit amount>".
I changed the Alfa transfer template {phone}T→{phone}R and the dest 1399→1313, but
the user first reverted BOTH then re-confirmed the DEST: current state (commit
4250ca3 on main, deployed) = template `{phone}T{amount}` (R reverted per user) +
**dest 1313** (user was emphatic it's 1313, same shortcode as Alfa recharge). The
transfer LETTER format is NOT yet resolved — user is investigating; the "Wrong format"
hint lists prefixes 03/70/71/76/79/81 (note: NO 78) and an R-letter, so the culprit
may be the phone-number formatting or a different mechanism, not just T-vs-R. Prod
BRIDGE_ env only sets ENABLED=true, TOUCH_MIN_BALANCE=0.5, ALFA_MIN_BALANCE=0.5 (all
other bridge knobs = config.go defaults). Samsung SM-S908N has only a Touch SIM, so
Alfa can't be tested on it.

**RECHARGE (recharge_line, scratch-card PIN → line) correct formats — user-provided,
wired 2026-07-08 (commit 5bdfccc):** Touch `*300*961{phone}*{card}#` (USSD; was
`*300*{phone}#{card}`), Alfa `*111*{code}*{phone}#` (USSD; WAS an SMS `{phone}R{code}`
to 1313 — wrong mechanism). rechargeAlfa switched from sendSms→sendUssd on the alfa
SIM, mirroring rechargeTouch (reply contains "fail" → ALFA_RECHARGE_PROVIDER_REJECTED
else _SUCCESS). {phone}=local 8-digit; Touch template literally prefixes 961, Alfa
does not. Note recharge (scratch card) ≠ transfer_credit (from SIM balance) — two
distinct flows. Open Q pending live test: Alfa recharge USSD success/fail reply text
(currently uses the "fail"-substring heuristic like Touch).

---

Written 2026-07-08 for a session dedicated to **debugging Alfa recharge**. Full
map (symptom buckets, end-to-end bridge_device path, Alfa config knobs, prod
facts to confirm, local repro, ranked suspects) is in repo root
**`HANDOFF-ALFA-DEBUG.md`** — start there.

Key first checks: is `BRIDGE_ENABLED=true` in prod (else all Alfa orders park,
never complete)? Do the real Alfa reply strings match `BRIDGE_SUCCESS_PATTERNS`?
Are the Alfa products actually `fulfillmentMode=bridge_device` with a valid
`Bridge` spec + variant `FaceValue`? Alfa flow lives in
`order/service.go` (fulfillBridge ~556), `order/phone.go`, `bridge/service.go`,
`config/config.go` (OperatorConfig Alfa knobs). See [[salehcard-mobile-bridge]]
and [[salehcard-prod-db-direct-access]].

Same session also: shipped i18n label fix to prod (Arabic-in-`en` → English via
untracked tool `api/cmd/inputlabels`), KYC back-button fix (main `6999939`),
built client APK at `app/build/app/outputs/flutter-apk/app-release.apk`.
