---
name: salehcard-business-tz-todo
description: Pending prod config — set BUSINESS_TZ=Asia/Beirut app setting on the API App Service
metadata: 
  node_type: memory
  type: project
  originSessionId: b10543d9-ea88-441e-bcf3-46ddbc11bab3
---

The admin dashboard's day-grained figures ("today" revenue/orders + their deltas, the daily revenue chart) bucket by a configurable timezone via the `BUSINESS_TZ` env var, added 2026-06-27 (see `api/internal/modules/order/repository.go` `businessLocation`). It **defaults to UTC**, so prod currently buckets "today" on UTC midnight, not Beirut.

**Why:** SalehCard operates in Lebanon (UTC+2/+3); orders placed before ~02:00–03:00 local land in the previous UTC day, so the "today" KPIs are wrong for the first hours of each business day until this is set.

**How to apply:** Add App Service app setting `BUSINESS_TZ=Asia/Beirut` on `salehcard-api` (RG `salehcard-prod`) and restart. Must be done in **Azure Cloud Shell** (the `az` CLI fails locally behind the corporate proxy — see [[salehcard-prod-deploy]]):
```
az webapp config appsettings set -g salehcard-prod -n salehcard-api --settings BUSINESS_TZ=Asia/Beirut
az webapp restart -g salehcard-prod -n salehcard-api
```
The tz database is embedded in the binary (`_ "time/tzdata"`), so it resolves regardless of the container OS. Deferred by user 2026-06-27 ("we will do it later").
