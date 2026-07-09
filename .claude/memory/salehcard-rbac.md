---
name: salehcard-rbac
description: "Admin-console RBAC — custom roles with domain×view/manage permissions, implicit Super Admin (adminRoleId==nil), perms ride the JWT; SHIPPED to main (PR #64, merge 25028f1) + auto-deployed to prod 2026-07-08, incl. 10 review-fix findings"
metadata: 
  node_type: memory
  type: project
  originSessionId: fdd31634-1787-4a8c-82fc-8c10c419e8a2
---

RBAC for the admin console — **SHIPPED to `main` (PR #64, merge `25028f1`) and auto-deployed to
prod on 2026-07-08** (CI + Deploy(prod) both green). Includes the 10 review-fix findings (F1–F10:
last-super-admin guard counts deleted, bulk-activate can't touch admins, raw-code reads need
inventory.manage, resend under orders.manage, boot-time domain-catalog validation, etc.).

Design (owner-confirmed choices):
- Permissions = `<domain>.view` / `<domain>.manage` over 17 domains (catalog is the single
  source of truth: `api/internal/modules/role/permissions.go`, served at
  `GET /api/admin/roles/permissions`). manage implies view (normalized server-side).
- **Super Admin is implicit**: an admin with `adminRoleId == nil` gets perms `["*"]` — no
  migration needed, all pre-RBAC admins are super admins. The `roles` collection holds only
  custom roles. Role CRUD + granting/revoking admin access are super-admin-only (never
  grantable), enforced in middleware (`auth.RequireSuperAdmin`) + in `user/admin.go updateRole`.
- Perms ride the JWT (`Claims.Perms`) and the auth-response user (`permissions`); role edits
  bite on the holder's next token refresh/login. Missing role at issue time → empty perms
  (fail-closed), resolver wired via `user.WithRolePerms` in `user/routes.go`.
- Enforcement: `auth.RequireDomain(name)` per module group in `server.go` — GET/HEAD →
  `.view`, else `.manage`. Module→domain map lives there (wallet→topups, code→inventory).
- LAST_ADMIN guard now counts **super admins** (`CountActiveSuperAdmins`); suspending/
  deleting/demoting admin accounts requires a super-admin actor.
- Frontend: `useCan()`/`useIsSuperAdmin()` in `admin/src/stores/auth.ts`; nav + routes gated
  (nav.ts `domain` field, `RequireDomain` wrapper, `firstAllowedRoute`); `/roles` page
  (super-admin-only, grp_system) with a domains×view/manage checkbox editor; role assignment
  select in Users → Role & access; **403 no longer logs out** (only 401 does) in api-client.

**Why:** owner asked for RBAC "on roles themselves, not pages" — permission sets attach to
role entities a super admin manages, so new pages need only a domain key, not new role logic.

**How to apply:** adding an admin module = pick a domain key, wrap it with `domain(...)` in
server.go, add the entry to `role/permissions.go` + `nav.ts` + router guard. Never grant
role management to a custom role. Related: [[salehcard-prod-deploy]] (merging to main ships
prod — existing prod admins become super admins automatically, zero migration).

Post-ship follow-up (do on prod when convenient): create the first real custom role in the
`/roles` editor and assign a second admin to it to exercise RBAC beyond the implicit super-admin
path. No migration or data backfill is required for the feature to work.
