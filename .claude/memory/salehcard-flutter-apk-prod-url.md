---
name: salehcard-flutter-apk-prod-url
description: "Customer Flutter APK must be built with the prod --dart-define API URL, or it hangs on a dead dev LAN IP (infinite loading)"
metadata: 
  node_type: memory
  type: project
  originSessionId: 1c61f1fb-4ffc-444f-a35e-b621ef91ff20
---

The `/app` customer Flutter app's API base URL defaults to a **hardcoded dev LAN IP**:
`app/lib/core/config/app_config.dart` → `API_BASE_URL` defaultValue
`http://192.168.10.170:8090/api/v1` (this dev box's Ethernet IP; overridable via
`--dart-define=API_BASE_URL=...`).

A plain `flutter build apk --release` bakes in that LAN address. On any real phone
(different network, laptop's API not running on :8090, or the IP unreachable from
Wi-Fi) every API call **hangs forever with no timeout** → the app sits on a loading
spinner and eventually throws
`Bad state: The provider FutureProvider<KycProfile> was disposed during loading state`.
Looks like an "internet/API/APK" problem but is neither — prod Azure is up; the app
just isn't pointed at it.

**Why:** the default was set for on-LAN device testing, never for distribution.

**How to apply:** ALWAYS build distributable / WhatsApp-sideload APKs against prod:
```
cd app && flutter build apk --release \
  --dart-define=API_BASE_URL=https://salehcard-api.azurewebsites.net/api/v1
```
Output: `app/build/app/outputs/flutter-apk/app-release.apk` (~58 MB, debug-key signed).
Prod API host = `https://salehcard-api.azurewebsites.net` (Go API; NOT the admin/static
web app URLs). Diagnose "stuck loading" via `adb logcat` filtered on `flutter` — a
disposed FutureProvider mid-load == unreachable base URL. See [[salehcard-mobile-bridge]]
(the bridge phone provisioning uses the same prod host) and [[salehcard-prod-deploy]].
