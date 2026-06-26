# Reusable per-module implementation prompt

Paste this at the start of a fresh module session, replacing `<MODULE>` (Shop | Orders | Wallet | Account | Browse | KYC).

---

Implement the **<MODULE>** module of the SalehCard Flutter app.

**Load context first:**
1. `CLAUDE.md` and `app/HANDOFF-DESIGN.md` (run/verify gotchas, conventions).
2. `app/design/MANIFEST.md` — find the `<MODULE>` section: screens, states, target file paths, routes, endpoints, ✅/🟡 status.
3. The plan at `C:\Users\user\.claude\plans\use-the-claude-design-mcp-gentle-hartmanis.md` (module decomposition + wiring).
4. The design source: `DesignSync get_file` on project `9c570d94-584a-41c6-9827-b072048e9688`, file `Phone<MODULE>.dc.html` (and `app/design/canvas/<MODULE>.dc.html` for the frame inventory).

**Clone these patterns exactly:**
- Data/domain/presentation layering from `features/catalog/` (datasource + `unwrap()`, `@JsonSerializable` DTO + `part '*.g.dart'` + `toEntity()` + `I18nString.fromJson`, `Either<Failure,T>` repo + `mapError`, layered `providers.dart`).
- Screen pattern from `features/auth/presentation/screens/login_screen.dart` (`ConsumerStatefulWidget`, `context.colors.*`, `AppLocalizations.of(context)`, `core/widgets/*`).
- Reuse shared widgets: `primary_cta`, `auth_text_field`, `country_code_box`, `step_dots`, `gradient_heading`, `product_chip`, plus foundation widgets `status_badge`, `empty_state`, `ledger_row`, `money_row`.

**Rules:**
- EN+AR strings via ARB + `flutter gen-l10n` (no hardcoded user-facing text). RTL-safe (`start`/`end`, `EdgeInsetsDirectional`). Light + dark via `context.colors`. Money via `formatUsd()`.
- DTOs: after editing, `dart run build_runner build --delete-conflicting-outputs`.
- ✅ endpoints: wire real HTTP per MANIFEST. 🟡: stub repo behind the real interface with `// TODO(backend)`.
- Add routes to `core/router/app_router.dart` (shell branch vs top-level per MANIFEST).
- Do NOT duplicate the Order entity/DTO (Orders reuses Shop's).

**Verify before done:**
```
cd C:\Users\user\salehcard\app
dart run build_runner build --delete-conflicting-outputs
flutter gen-l10n && flutter analyze && flutter test
```
Then on device: API on :8090, `adb reverse tcp:8090 tcp:8090`, `flutter run --dart-define=API_BASE_URL=http://127.0.0.1:8090/api/v1`, login `customer@salehcard.local`/`password123`. Screenshot each screen × EN/AR × light/dark × each state; confirm ✅ endpoints return 200 (not 401), RTL + dark correct, 🟡 stubs render all states. Commit `app/` only when green.
