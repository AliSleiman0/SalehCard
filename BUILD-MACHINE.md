# This box as an Android build machine — 2026-07-24

Set up overnight. **Verified: it builds the app.** `flutter build apk --debug` produced
`app/build/app/outputs/flutter-apk/app-debug.apk` (157.3 MB, exit 0).

## What's installed

| Component | Version | Location |
| --- | --- | --- |
| Flutter | 3.44.8 stable (Dart 3.12.2 — matches `pubspec` `sdk: ^3.12.2`) | `C:\dev\tools\flutter` |
| JDK | Temurin 17 | `C:\dev\tools\jdk17` |
| Android SDK | cmdline-tools, platform-tools, platforms 35 + 36, build-tools 36.0.0 | `%LOCALAPPDATA%\Android\Sdk` |
| NDK | 28.2.13676358 (r28c) | `%LOCALAPPDATA%\Android\Sdk\ndk\28.2.13676358` |
| CMake | 3.22.1 (auto-installed by AGP) | `%LOCALAPPDATA%\Android\Sdk\cmake` |
| Gradle | 9.1.0 distribution seeded into the wrapper cache | `~\.gradle\wrapper\dists` |
| Emulator | installed; **system image still downloading** | `%LOCALAPPDATA%\Android\Sdk\emulator` |

`JAVA_HOME`, `ANDROID_HOME`, `ANDROID_SDK_ROOT` and the PATH entries are persisted at
**User** scope, so a fresh terminal picks them up.

## Build commands

```powershell
cd C:\dev\SalehCard\app
flutter build apk --debug   --dart-define=API_BASE_URL=http://10.0.2.2:8090/api/v1   # local API from emulator
flutter build appbundle     --dart-define=API_BASE_URL=https://salehcard-api.azurewebsites.net/api/v1
```

Never omit the dart-define — the default in `app_config.dart:13` is a dev LAN IP.

## The one blocker: signing

`flutter build appbundle` gets **all the way through compile + R8 + bundle assembly** and
fails only at `:app:signReleaseBundle` with a `NullPointerException` — the null `storeFile`
from the missing `android/key.properties`. An **unsigned** 146.5 MB bundle is left at
`app/build/app/intermediates/intermediary_bundle/release/.../intermediary-bundle.aab`
(not uploadable — Play requires the signature).

To finish: drop `upload-keystore.jks` + `key.properties` (storeFile / storePassword /
keyAlias=upload / keyPassword) into `app/android/`, both gitignored. Nothing else is missing.

## Gotchas hit (all cost real time — worth knowing)

1. **PowerShell `Invoke-WebRequest` stalls at 0 bytes** on large downloads here. It sat 38
   minutes on the Flutter zip without writing a byte. **Use `curl.exe`** — it streams and
   resumes (`-C -`).
2. **`sdkmanager` stalls the same way on big packages.** It installed platform-tools,
   android-36 and build-tools in ~1 minute each, then failed the 1.4 GB system image
   **6 times** with zero bytes moved. Big SDK artifacts must be fetched with curl and
   unpacked by hand.
3. **The Gradle wrapper also stalls** fetching `gradle-9.1.0-all.zip`. Fix: curl the zip
   into `~\.gradle\wrapper\dists\gradle-9.1.0-all\<hash>\` and delete the `.part`/`.lck`.
4. **The NDK is required** (`ndkVersion` is declared in `build.gradle.kts`) and AGP tries to
   auto-install it — 1 GB through the stalling downloader. Fetched
   `android-ndk-r28c-windows.zip` with curl and extracted it to `ndk\28.2.13676358`;
   AGP validates via `source.properties`, so no `package.xml` is needed.
5. **Git-Bash `tar` can't take `C:/...` paths** ("Cannot connect to C: resolve failed") —
   use PowerShell's `tar.exe` with backslash paths, or `/c/...` in bash.
6. Flutter hides Gradle errors; run `android\gradlew.bat assembleDebug --console=plain`
   to see the real failure.
7. `flutter doctor` reports Visual Studio "incomplete" — irrelevant, that's the Windows
   desktop target only.

## Local stack (running)

- Mongo already listening on 27017; `api/.env` present (`PORT=8090`, `ENV=development`,
  `SMS_PROVIDER=log` → OTP codes appear in the API log rather than being sent).
- Seeded via `go run ./cmd/seed`; API running from `go run ./cmd/server`, verified
  `GET /api/v1/products` → 200 with 3 products.
