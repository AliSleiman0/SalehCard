---
name: salehcard-flutter-emulator-run
description: How to run the /app Flutter client on the Android emulator against the local dev API — the exact recipe plus the startup-lock + output-capture gotchas
metadata: 
  node_type: memory
  type: feedback
  originSessionId: b10543d9-ea88-441e-bcf3-46ddbc11bab3
---

Running the SalehCard `/app` Flutter client on the Android emulator against the local Go API. The canonical recipe is in `app/HANDOFF-DESIGN.md`; this captures what actually bit me so I don't re-thrash next time. See [[salehcard-flutter-mobile]] and [[salehcard-dev-env]].

**The recipe (do exactly this, sequentially):**
1. Make sure the Go API is up on `:8090` (`curl localhost:8090/health` → 200).
2. Launch the emulator **directly via the binary**, NOT `flutter emulators --launch`:
   `& "$env:LOCALAPPDATA\Android\Sdk\emulator\emulator.exe" -avd Medium_Phone_API_36.1` (Start-Process, detached). Then `adb wait-for-device` + poll `getprop sys.boot_completed`==1.
3. `adb -s emulator-5554 reverse tcp:8090 tcp:8090` (re-run after any reconnect) — tunnels device `127.0.0.1:8090` → host.
4. ONE flutter invocation: `flutter run --dart-define=API_BASE_URL=http://127.0.0.1:8090/api/v1 -d emulator-5554` (from `app/`). adb at `%LOCALAPPDATA%\Android\Sdk\platform-tools\adb.exe`; flutter at `C:\src\flutter\bin\flutter.bat`.

**Why:** the app's API base URL is a compile-time `String.fromEnvironment('API_BASE_URL', default 'http://192.168.10.170:8090/api/v1')` (`app/lib/core/config/app_config.dart`) — a stale ex-dev-machine LAN IP. "Couldn't load products" = the installed build is on that default. Must rebuild with `--dart-define`; there is no runtime override. The dev pattern is `adb reverse` + `127.0.0.1` (NOT `10.0.2.2` — that works too but reverse+127.0.0.1 is what the handoff documents).

**Gotchas that cost me ~an hour:**
- **Startup-lock deadlock:** any stray flutter process (e.g. a leftover `flutter analyze` from an editor/LSP) holds `C:\src\flutter\bin\cache\lockfile`, so every `flutter run`/`emulators` hangs on "Waiting for another flutter command to release the startup lock". The lock holder's worker is **`dartvm.exe`**, which `Get-Process -Name dart` does NOT catch — sweep `dart,dartvm,java` and delete the lockfile. Run flutter commands strictly **one at a time**; concurrent ones self-deadlock. `flutter emulators --launch` itself sometimes hangs holding the lock without bringing the window up — hence launching the emulator binary directly.
- **`flutter run` output never flushes to a backgrounded file** (0 bytes even while building/running). Don't gate on the output file. Verify state via adb instead: `adb shell pidof com.salehcard.salehcard_app` (app running), `adb shell screencap -p /sdcard/_s.png` + `pull` + downscale-to-460 + Read the PNG (emulator 1080×2400 → real coord = downscaled × 2.348). Bring the app to foreground with `adb shell monkey -p com.salehcard.salehcard_app -c android.intent.category.LAUNCHER 1`.
- `flutter run` is backgrounded (no stdin → no `r`/`R` hot reload); to reload after a change, kill `dart`+`dartvm` and re-run.
