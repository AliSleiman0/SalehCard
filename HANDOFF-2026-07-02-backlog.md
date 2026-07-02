# Handoff — 2026-07-02 (backlog session: BL-1, BL-2 done → next is BL-4)

Session context for the Flutter customer app (`/app`). Two backlog items shipped
this session; **the next session should start BL-4**. Manual end-to-end testing
of BL-1/BL-2 was intentionally deferred by the user (see "Not yet tested").

Read `BACKLOG.md` for the full item list. This file is the bridge.

---

## Done this session (code complete, static-verified)

### BL-1 — Wallet refresh loop ✅
Pull-to-refresh + 45s foreground polling so admin-approved credit / completed
orders appear without navigating away.
- **`app/lib/features/wallet/presentation/screens/wallet_screen.dart`** — now a
  `ConsumerStatefulWidget`; `RefreshIndicator` (refreshes `walletProvider` +
  `topUpRequestsProvider`, awaits the refetch); 45s `Timer.periodic` started in
  `initState`, paused on background via `WidgetsBindingObserver`, immediate
  refresh on resume; `AlwaysScrollableScrollPhysics`.
- **`orders_list_screen.dart`** — `RefreshIndicator` on the list; empty state made
  pullable (wrapped in full-height `SingleChildScrollView`/`ConstrainedBox`).
- **`order_detail_screen.dart`** — `RefreshIndicator` on `orderDetailProvider(id)`.
- **`topup_screen.dart`** — `RefreshIndicator` on the request history
  (`topUpRequestsProvider` + `walletProvider`).
- Relied on `.when`'s default `skipLoadingOnRefresh: true` (no spinner flash on
  poll). No API/provider changes.

### BL-2 — Pending top-up banner on Wallet ✅
Compact banner explaining an unchanged balance (funding is admin-approved).
- **New l10n keys** in `app/lib/core/i18n/arb/app_en.arb` + `app_ar.arb`:
  `walletPendingTitle` (ICU plural on `count`) and `walletPendingHint`.
  Regenerated with `flutter gen-l10n` (getters live in `app_localizations*.dart`).
- **`wallet_screen.dart`** — watches `topUpRequestsProvider`; new private
  `_PendingTopUpBanner` (clock icon + plural count + hint + summed
  `formatUsd(total)` + chevron), inserted after the action-button row; guarded by
  `maybeWhen(..., orElse: SizedBox.shrink)` + empty-pending check so it never
  blocks the wallet. Taps → `/wallet/topup`. Decisions: **pending-only**, **single
  summary banner** (per user).

**Verify status:** `flutter gen-l10n` + `flutter analyze` (No issues) +
`flutter test` (15/15 pass) all green. `dart format` applied to touched files.

### Not yet tested (user deferred — do this before considering BL-1/BL-2 closed)
Needs running API + Mongo + emulator/device:
1. File a top-up → Wallet shows pending banner (count + total) → approve in
   `/admin` → pull-to-refresh / 45s poll clears banner and updates balance.
2. Second pending request → banner switches to plural + summed total.
3. Order `processing → completed` reflected via pull-to-refresh on Order detail.
4. Background >45s → no polling; resume → one immediate refresh.
5. Arabic locale: both new strings + RTL plural render correctly.

---

## NEXT: BL-4 — Home screen search + bell wiring (Flutter)

**Goal:** `home_screen.dart` wires the search bar and the notification bell to a
"coming soon" snackbar, but fully working `/search` and `/notifications` screens
already exist and are already wired from the Browse tab. Two one-line rewires
(+ a small cleanup). Both routes are already registered in
`app/lib/core/router/app_router.dart` (`/search` line 103, `/notifications` line
109), and `home_screen.dart` already imports `go_router`.

**Working precedent to copy:** `browse/presentation/screens/categories_screen.dart`
— search box `onTap: () => context.push('/search')` (~line 64) and the bell
`IconButton(onPressed: () => context.push('/notifications'))` (~line 34-38).

**Edits — `app/lib/features/home/presentation/screens/home_screen.dart`:**
1. **Search bar** (line ~64): `_SearchBar(hint: l10n.searchHint, onTap: _comingSoon)`
   → `onTap: () => context.push('/search')`.
2. **Notification bell** (lines ~192-197): the `IconButton` whose
   `onPressed: () => ScaffoldMessenger...comingSoon` → `onPressed: () =>
   context.push('/notifications')`.
3. **Cleanup:** after (1), the private `_comingSoon()` method (lines ~30-34) is
   the *only* remaining local user — once the bell is also rewired it becomes
   unused → `flutter analyze` will flag `unused_element`. Delete the
   `_comingSoon()` method.
   - **Keep the `comingSoon` l10n key** — still used by the scan FAB in
     `app/lib/features/shell/presentation/app_shell.dart` (`_comingSoon`,
     lines ~20/60). Do NOT remove the key.

**Verify:** `cd app && flutter analyze && flutter test`. Then manual: from Home,
tap the search bar → `/search` opens; tap the bell → `/notifications` opens.

---

## After BL-4 (small Flutter items, good follow-ons)
- **BL-5** — purge stale "instant" copy (`promoSubtitle`, `deliveredInstantly`,
  notification stub bodies) + delete orphaned l10n keys (`payCardTitle`,
  `payCardSub`, `payUsdtTitle`, `payUsdtSub`, `topUpSuccess`) from en+ar arb and
  regenerate. **Note:** BL-2 did NOT touch `topUpSuccess` — it's still orphaned
  and is on BL-5's delete list; leave it for BL-5.
- **BL-3** — admin work-queue signals (backend + admin console; not Flutter).

## Working notes / gotchas
- l10n is generated (`app/l10n.yaml`, `generate: true`): edit both `app_en.arb`
  and `app_ar.arb`, then `flutter gen-l10n` regenerates the three
  `app_localizations*.dart` files. ICU placeholder messages need an
  `@key` placeholders block (see `topUpSuccess` / `walletPendingTitle`).
- Verify gate for app changes: `cd app && flutter analyze && flutter test`
  (run `flutter gen-l10n` first if you touched arb files). Run `dart format` on
  touched files.
- Nothing committed this session (repo default is commit only when asked). The
  four BL-1 files + `wallet_screen.dart` (BL-2) + the two arb files + regenerated
  `app_localizations*.dart` are uncommitted working-tree changes.
