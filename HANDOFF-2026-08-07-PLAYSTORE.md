# HANDOFF — FlashCash Global Play Store, closed testing setup (2026-08-07)

Continues from `HANDOFF-2026-07-25-PLAYSTORE.md` (now superseded for the account/package
questions it left open) and `[[salehcard-playstore-org-account-path]]` memory. This session
did the full org-account resubmission end-to-end in Play Console via Claude-in-Chrome and
figured out why "Send app for review" wouldn't enable. **Nothing has been submitted for
review yet** — that's the next session's job, and it's a small, well-understood step.

---

## TL;DR

| Item | State |
| --- | --- |
| New applicationId `com.flashcashglobal.app` (old `flashcash.global` permanently locked to a discarded account) | ✅ shipped, commit `7eb3eba` |
| Play Console app created under **org account "Entrizo"** | ✅ done |
| Signed release AAB uploaded, live on **Internal testing** | ✅ release 4 (1.0.3), "Available to internal testers," 19,094 devices |
| All 10 App content declarations (Advertising ID, Data safety, Target audience, Sign-in details, Financial features, Government apps, Content ratings, Ads, Privacy policy) | ✅ all "actioned," no issues |
| Store listing (en-US) | ✅ status "Ready to send for review" |
| **"Send app for review" button on Publishing overview** | 🔴 stays disabled — **root cause found** (see below), fix not yet applied |
| Closed testing track "Alpha" | 🟡 exists but **empty** — no country/region, no testers, no release |
| Reviewer sign-in test account | 🟡 currently the **live prod admin account** — should be swapped before actual submission |

---

## Why "Send app for review" is disabled (found this session)

Spent a long time assuming a missing declaration or a Play Console sync bug — ruled both out
completely (every App content item shows actioned, dashboard checklist is empty, waited 20+
min and reloaded repeatedly). Filed a Play Console AI-assisted support ticket
(`Help → Create support ticket → App publishing`); the AI assistant gave the real answer and
the ticket auto-resolved (visible under Help → Your support tickets → "App publishing" —
"Resolved by AI assistant"):

> The "Send app for review" button specifically processes changes for tracks that require a
> full review cycle before public distribution (Closed testing, Open testing, Production).
> **Internal testing releases never go through Google review** — that's why the button stays
> disabled even though the Internal testing release is fully live. You need a release on a
> reviewable track before this button does anything.

Also confirmed: since this is an **organization account**, it is **not** subject to the
mandatory 14-day/12-tester closed-testing gate that blocks new *personal* accounts from
reaching Production — so there's no forced waiting period once the steps below are done.

---

## Exact next steps

1. **Play Console → Test and release → Testing → Closed testing → "Alpha" track → Manage
   track.** URL: `https://play.google.com/console/u/3/developers/8912936751869131931/app/4975962755268243214/closed-testing`
   Track ID seen this session: `4699404763906166543`.
2. **Select countries and regions** for the track (not yet chosen — pick whichever markets
   the client wants to test in first, at minimum Lebanon).
3. **Select testers** — this track needs its **own** tester list (email list or Google Group),
   separate from the Internal testing tester list (`clashroyale084@gmail.com`,
   `sleimana181@gmail.com`, `osama.abouzaid@oz-consultants.com`) set up earlier.
4. **Create a release on this track.** Two options:
   - Promote the existing live Internal testing release (4, 1.0.3) directly — on the Internal
     testing release page there's a **"Promote release"** dropdown with **Closed testing**,
     Open testing, Production as options. No AAB re-upload needed.
   - Or create a new release from scratch and re-upload the AAB at
     `~/Downloads/flashcash-release/2026-08-07/flashcash-global-1.0.3+4-com.flashcashglobal.app.aab`.
   Promoting the existing release is simpler and is the recommended path.
5. Once a release exists on Closed testing, go back to **Publishing overview**
   (`https://play.google.com/console/u/3/developers/8912936751869131931/app/4975962755268243214/publishing`)
   and confirm **"Send app for review"** is now enabled. Click it.

## Before actually submitting — swap the reviewer test account

The Sign-in details declaration currently points reviewers at phone `+96178991778` with a
password that turned out to be the **live prod `admin@salehcard.com` account** (full
admin/wallet access) — not a disposable reviewer account. This was only meant as a login
*verification* during setup, not what should ship to Google's reviewer. **Create a dedicated
non-admin test account** (customer role, demo wallet balance, KYC pre-approved so the reviewer
isn't blocked by the purchase-gate) and update the Sign-in details declaration
(`App content → Sign-in details`) before clicking "Send app for review." Remember the UI
gotcha: filling the modal and clicking "Add" only stages it locally — you must **also** click
the page-level "Save" button below the whole form for it to persist.

---

## Reference

- Org account: **Entrizo**, account ID `8912936751869131931`.
- App: **FlashCash Global**, app ID `4975962755268243214`, package `com.flashcashglobal.app`.
- Publishing overview: `https://play.google.com/console/u/3/developers/8912936751869131931/app/4975962755268243214/publishing`
- App content: `https://play.google.com/console/u/3/developers/8912936751869131931/app/4975962755268243214/app-content/overview`
- Dashboard: `https://play.google.com/console/u/3/developers/8912936751869131931/app/4975962755268243214/app-dashboard`
- Advertising ID declaration answered **No** — app only ships `firebase_core` +
  `firebase_messaging` (push), no ads/analytics SDK that reads it.
- A general-purpose Play Store readiness audit prompt (covers the whole process, rules, and UI
  gotchas from this session) was written into chat this session for use on a different app —
  ask the owner if they want it saved to a file too, it wasn't persisted anywhere.
- Still deliberately open, not blocking: Google API-key restriction for FCM push on the new
  package (`DEVOPS-TODO.md` §22 — needs `gcloud`, not on this box, risky to hand-edit since it
  gates the live prod app's key too).

## Environment note

All Play Console work this session was done via **Claude-in-Chrome** browser automation.
Known flakiness: `Page.captureScreenshot` times out often (~30s) on this heavy Flutter-web
app — retry once. Viewport width can drift between screenshots, causing coordinate clicks to
miss — re-screenshot immediately before a critical click, or use `read_page`/`find` refs.
