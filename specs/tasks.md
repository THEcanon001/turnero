# Task Breakdown: Turnero App

> Organized by User Story phases. Tasks marked [P] can be parallelized. Dependencies noted explicitly.

---

## Phase 0: Project Setup

> **Prerequisite for all phases. Go backend follows Package Oriented Design (POD).**

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T0-01 | Initialize Go module, POD project skeleton | `go.mod`, `cmd/server/main.go`, `Makefile`, POD directory structure | - | - |
| T0-02 | Set up PostgreSQL via docker-compose for dev | `docker-compose.yml` | [P] | - |
| T0-03 | Implement config loading from env vars | `internal/platform/config/config.go` | [P] | - |
| T0-04 | Set up database connection pool (pgxpool) | `internal/platform/database/database.go` | - | T0-01, T0-02 |
| T0-05 | Set up migration framework (golang-migrate) | `internal/platform/database/migrations/` | - | T0-04 |
| T0-06 | Write initial migration (full schema) | `internal/platform/database/migrations/001_initial.up.sql`, `001_initial.down.sql` | - | T0-05 |
| T0-07 | Set up Chi router with middleware stack | `cmd/server/main.go`, `internal/platform/middleware/` | [P] | T0-01 |
| T0-08 | Implement logging middleware (slog) | `internal/platform/middleware/logging.go` | [P] | T0-07 |
| T0-09 | Implement panic recovery middleware | `internal/platform/middleware/recovery.go` | [P] | T0-07 |
| T0-10 | Implement CORS middleware | `internal/platform/middleware/cors.go` | [P] | T0-07 |
| T0-11 | Initialize Expo project with expo-router | `turnero-app/` (frontend root) | [P] | - |
| T0-12 | Set up NativeWind (Tailwind) for styling | `turnero-app/tailwind.config.js` | - | T0-11 |
| T0-13 | Set up TanStack Query + Zustand | `turnero-app/lib/`, `turnero-app/stores/` | - | T0-11 |
| T0-14 | Create API client with base URL + auth headers | `turnero-app/lib/api.ts` | - | T0-11 |
| T0-15 | Create health endpoint | `cmd/server/main.go` (inline or small handler) | [P] | T0-07 |

**Checkpoint:** Go server starts with POD structure, connects to PostgreSQL, runs migrations, responds to `/health`. Expo app builds and runs.

---

## Phase 1: Auth + Provider Registration (US-P1-01)

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T1-01 | Implement JWT helper (generate, validate, claims) | `pkg/jwt/jwt.go` | [P] | T0-01 |
| T1-02 | Implement auth service (bcrypt hash/verify, JWT sign) | `internal/auth/service.go` | - | T1-01 |
| T1-03 | Implement provider model + auth model + validation | `internal/provider/model.go`, `internal/auth/model.go` | [P] | T0-01 |
| T1-04 | Implement provider repository (Create, GetByEmail, GetBySlug) | `internal/provider/repository.go` | - | T0-06, T1-03 |
| T1-05 | Implement refresh token repository | `internal/auth/repository.go` | [P] | T0-06 |
| T1-06 | Implement auth handler (register, login, refresh, logout) | `internal/auth/handler.go` | - | T1-02, T1-04, T1-05 |
| T1-07 | Implement JWT auth middleware | `internal/platform/middleware/auth.go` | - | T1-01 |
| T1-08 | Wire auth routes in router | `cmd/server/main.go` | - | T1-06, T1-07 |
| T1-09 | [Frontend] Create login screen | `turnero-app/app/(auth)/login.tsx` | [P] | T0-14 |
| T1-10 | [Frontend] Create registration screen | `turnero-app/app/(auth)/register.tsx` | [P] | T0-14 |
| T1-11 | [Frontend] Implement auth context + SecureStore token storage | `turnero-app/lib/auth.ts`, `turnero-app/hooks/useAuth.ts` | - | T1-09 |
| T1-12 | [Frontend] Create root layout with auth check | `turnero-app/app/_layout.tsx` | - | T1-11 |

**Checkpoint:** Provider can register, login, receive JWT, and access authenticated endpoints. Frontend login/register flow works. Errors follow `auth.handler: auth.service: ...` convention.

---

## Phase 2: Services + Schedules (US-P1-02, US-P1-03)

> Note: "service" (business service like "Haircut") lives in `internal/provider/` since it's a sub-entity of provider. Could also be its own domain package if it grows.

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T2-01 | Implement service model + validation | `internal/provider/model.go` (add Service type) | [P] | - |
| T2-02 | Implement service repository (CRUD) | `internal/provider/repository.go` (add service methods) | - | T0-06, T2-01 |
| T2-03 | Implement service handler (CRUD endpoints) | `internal/provider/handler.go` | - | T2-02, T1-07 |
| T2-04 | Implement employee model | `internal/employee/model.go` | [P] | - |
| T2-05 | Implement employee repository | `internal/employee/repository.go` | - | T0-06, T2-04 |
| T2-06 | Auto-create "self" employee for PF providers on registration | `internal/auth/service.go` (update) | - | T2-05, T1-02 |
| T2-07 | Implement schedule model | `internal/schedule/model.go` | [P] | - |
| T2-08 | Implement schedule repository (CRUD + exceptions) | `internal/schedule/repository.go` | - | T0-06, T2-07 |
| T2-09 | Implement schedule handler (weekly config + exceptions) | `internal/schedule/handler.go` | - | T2-08, T1-07 |
| T2-10 | Wire service + schedule routes | `cmd/server/main.go` | - | T2-03, T2-09 |
| T2-11 | [Frontend] Services list + create/edit screen | `turnero-app/app/(provider)/services/` | [P] | T0-14 |
| T2-12 | [Frontend] Weekly schedule config screen | `turnero-app/app/(provider)/schedule/index.tsx` | [P] | T0-14 |
| T2-13 | [Frontend] Schedule exceptions screen | `turnero-app/app/(provider)/schedule/exceptions.tsx` | [P] | T0-14 |

**Checkpoint:** Provider can create services, configure weekly schedule, and add exceptions. Slots are implicitly defined.

---

## Phase 3: Availability + Booking (US-P1-04, US-P1-05, US-P1-07)

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T3-01 | Implement availability service (slot calculation) | `internal/appointment/service.go` | - | T2-08 |
| T3-02 | Implement appointment model + validation | `internal/appointment/model.go` | [P] | - |
| T3-03 | Implement appointment repository (Create, Get, List, UpdateStatus) | `internal/appointment/repository.go` | - | T0-06, T3-02 |
| T3-04 | Implement availability handler (public: provider profile, employee slots) | `internal/appointment/handler.go` | - | T3-01, T1-04 |
| T3-05 | Implement appointment handler (client: create, view, cancel) | `internal/appointment/handler.go` | - | T3-03, T3-01 |
| T3-06 | Implement search handler (fuzzy search by name/slug) | `internal/search/handler.go` | [P] | T1-04 |
| T3-07 | Implement optimistic locking on appointment creation (UNIQUE constraint handling) | `internal/appointment/repository.go` (update) | - | T3-03 |
| T3-08 | Wire public + appointment routes | `cmd/server/main.go` | - | T3-04, T3-05, T3-06 |
| T3-09 | [Frontend] Client home + search screen | `turnero-app/app/(client)/index.tsx`, `search.tsx` | [P] | T0-14 |
| T3-10 | [Frontend] Provider profile screen | `turnero-app/app/(client)/provider/[slug].tsx` | - | T3-09 |
| T3-11 | [Frontend] Booking flow (service → employee → date → time → confirm) | `turnero-app/app/(client)/book/[slug].tsx` | - | T3-10 |
| T3-12 | [Frontend] DayPicker + SlotGrid components | `turnero-app/components/calendar/` | [P] | T0-12 |
| T3-13 | [Frontend] "Mis turnos" list + detail screen | `turnero-app/app/(client)/appointments/` | - | T3-11 |
| T3-14 | [Frontend] Cancel/reschedule flow on appointment detail | `turnero-app/app/(client)/appointments/[id].tsx` | - | T3-13 |
| T3-15 | [Frontend] Provider agenda view (day/week) | `turnero-app/app/(provider)/agenda/` | [P] | T0-14 |
| T3-16 | [Frontend] Appointment detail (complete, no-show, cancel, WhatsApp) | `turnero-app/app/(provider)/agenda/[id].tsx` | - | T3-15 |
| T3-17 | [Frontend] AgendaView component | `turnero-app/components/calendar/AgendaView.tsx` | [P] | T0-12 |
| T3-18 | Implement provider GET /v1/appointments (authenticated, with filters) | `internal/appointment/handler.go` (update) | - | T3-05 |

**Checkpoint:** Full booking flow works end-to-end. Client can search, view provider, book, cancel, reschedule. Provider can view agenda and manage appointments.

---

## Phase 4: QR + WhatsApp (US-P1-05, US-P1-06, US-P3-03)

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T4-01 | Implement QR service (go-qrcode, generate PNG) | `internal/qr/service.go` | [P] | - |
| T4-02 | Implement QR handler (GET /provider/me/qr) | `internal/qr/handler.go` | - | T4-01 |
| T4-03 | Configure Nginx to serve QR images from /qr/ path | Nginx config | - | T4-01 |
| T4-04 | Implement WhatsApp deep link helper | `internal/appointment/handler.go` (update — add whatsapp_link to responses) | [P] | - |
| T4-05 | [Frontend] QR screen (generate, download, share) | `turnero-app/app/(provider)/qr.tsx` | [P] | T0-14 |
| T4-06 | [Frontend] QR scanner (expo-camera) | `turnero-app/app/(client)/scan.tsx` | [P] | T0-11 |
| T4-07 | [Frontend] WhatsApp button component + deep link helper | `turnero-app/lib/whatsapp.ts` | [P] | T0-11 |

**Checkpoint:** Provider can generate/share QR. Client can scan QR and land on provider profile. WhatsApp links work throughout the app.

---

## Phase 5: Business (PJ) + Employees (US-P2-01 to US-P2-05, US-P2-11, US-P2-12)

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T5-01 | Implement employee handler (CRUD, invitation codes) | `internal/employee/handler.go` | - | T2-05, T1-07 |
| T5-02 | Implement invitation code generation + validation | `internal/employee/repository.go` (add invitation methods) | - | T0-06 |
| T5-03 | Implement POST /v1/join (accept invitation) | `internal/auth/handler.go` (update) | - | T5-02, T1-02 |
| T5-04 | Implement employee_services repository (assign services to employees) | `internal/employee/repository.go` (update) | - | T2-02, T2-05 |
| T5-05 | Implement "any available" employee assignment logic | `internal/appointment/service.go` (update) | - | T3-01 |
| T5-06 | Implement business hours enforcement on employee schedules | `internal/schedule/handler.go` (update) | - | T2-09 |
| T5-07 | Implement provider settings handler | `internal/provider/handler.go` (update) | - | T1-04 |
| T5-08 | Add employee-level auth (role-based access in middleware) | `internal/platform/middleware/auth.go` (update) | - | T1-07 |
| T5-09 | Wire employee routes | `cmd/server/main.go` | - | T5-01 |
| T5-10 | [Frontend] Employee list + invite screen | `turnero-app/app/(provider)/employees/` | [P] | T0-14 |
| T5-11 | [Frontend] Employee detail + assign services | `turnero-app/app/(provider)/employees/[id].tsx` | - | T5-10 |
| T5-12 | [Frontend] Join business flow (accept invitation code) | `turnero-app/app/(employee)/join.tsx` | [P] | T0-14 |
| T5-13 | [Frontend] Employee layout + own agenda | `turnero-app/app/(employee)/` | - | T5-12 |
| T5-14 | [Frontend] Provider settings screen | `turnero-app/app/(provider)/settings.tsx` | [P] | T0-14 |

**Checkpoint:** Full PJ (business) flow works. Admin can create business, invite employees, assign services. Employees can join, set availability, manage their agenda. Clients can book with "any" employee.

---

## Phase 6: Cancel/Reschedule + Block Days (US-P2-06 to US-P2-10, US-P2-13)

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T6-01 | Implement client cancel with time restriction | `internal/appointment/handler.go` (update) | - | T3-05 |
| T6-02 | Implement client reschedule (atomic slot swap) | `internal/appointment/handler.go` (update) | - | T3-05 |
| T6-03 | Implement provider cancel (no restriction) | `internal/appointment/handler.go` (update) | - | T3-05 |
| T6-04 | Implement batch cancel when blocking days (list affected, cancel all) | `internal/schedule/handler.go` (update) | - | T2-09, T3-03 |
| T6-05 | Implement employee deactivation with appointment handling | `internal/employee/handler.go` (update) | - | T5-01, T3-03 |
| T6-06 | Implement appointment reassignment (admin moves turno to another employee) | `internal/appointment/handler.go` (update) | - | T3-05 |
| T6-07 | Implement walk-in (manual) appointment creation | `internal/appointment/handler.go` (update) | - | T3-05 |

**Checkpoint:** All cancel/reschedule/block flows work. Admin can reassign appointments between employees.

---

## Phase 7: Push Notifications (US-P3-01, US-P3-02, US-P3-05, US-P3-07)

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T7-01 | Implement push token repository | `internal/notification/repository.go` | [P] | T0-06 |
| T7-02 | Implement push token handler (register/deregister) | `internal/notification/handler.go` | - | T7-01 |
| T7-03 | Implement notification service (FCM HTTP API sender) | `internal/notification/service.go` | - | T7-01 |
| T7-04 | Implement reminder worker (cron: check upcoming appointments every 5min) | `internal/notification/worker.go` | - | T7-03, T3-03 |
| T7-05 | Send push on appointment created/cancelled/rescheduled | `internal/appointment/handler.go` (update) | - | T7-03 |
| T7-06 | Implement weekly summary push (cron: Monday 9AM) | `internal/notification/worker.go` (update) | - | T7-03 |
| T7-07 | [Frontend] Register FCM token on app launch | `turnero-app/lib/notifications.ts` | [P] | T0-11 |
| T7-08 | [Frontend] Handle push notification tap (navigate to appointment) | `turnero-app/lib/notifications.ts` (update) | - | T7-07 |

**Checkpoint:** Push notifications work end-to-end. Reminders sent 24h and 2h before appointments. Weekly summary for providers.

---

## Phase 8: Statistics (US-P3-04, US-P3-06)

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T8-01 | Implement stats queries in repository (completed, cancelled, no-show, by employee) | `internal/appointment/repository.go` (update) | - | T3-03 |
| T8-02 | Implement stats handler | `internal/provider/handler.go` (update) | - | T8-01 |
| T8-03 | [Frontend] Stats screen (PF) | `turnero-app/app/(provider)/stats.tsx` | [P] | T0-14 |
| T8-04 | [Frontend] Stats screen (PJ — consolidated + per employee) | `turnero-app/app/(provider)/stats.tsx` (update) | - | T8-03 |

**Checkpoint:** Provider sees monthly stats. Admin PJ sees consolidated + per-employee breakdown.

---

## Phase 9: Billing + Monetization (US-P4-01, US-P4-02, US-P4-03)

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T9-01 | Implement billing service (monthly count, cap calculation) | `internal/billing/service.go` | - | T3-03 |
| T9-02 | Implement billing repository (usage tracking, transaction history) | `internal/billing/repository.go` | [P] | T0-06 |
| T9-03 | Implement billing worker (cron: monthly cycle, count completions) | `internal/billing/worker.go` | - | T9-01, T9-02 |
| T9-04 | Increment billing counter on appointment.completed | `internal/appointment/handler.go` (update) | - | T9-01 |
| T9-05 | Implement billing handler (GET current usage, GET history) | `internal/billing/handler.go` | - | T9-02, T1-07 |
| T9-06 | Implement grace period logic (3 months unpaid → restrict) | `internal/billing/service.go` (update) | - | T9-01 |
| T9-07 | Send push on free tier exceeded | `internal/appointment/handler.go` (update) | - | T7-03, T9-01 |
| T9-08 | Send push on monthly invoice | `internal/billing/worker.go` (update) | - | T7-03, T9-03 |
| T9-09 | [Frontend] Billing screen (usage bar, estimated cost, history) | `turnero-app/app/(provider)/billing.tsx` | [P] | T0-14 |

**Checkpoint:** Billing system tracks completed appointments per month. Provider sees usage + cost. Cap at ~$10/month.

---

## Phase 10: Observability + DevOps

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T10-01 | Implement Prometheus metrics (HTTP + business) | `internal/platform/middleware/metrics.go` | [P] | T0-07 |
| T10-02 | Expose /metrics endpoint | `cmd/server/main.go` | - | T10-01 |
| T10-03 | Implement metrics updater worker (business gauges every 5min) | `internal/worker/metrics.go` | - | T10-01 |
| T10-04 | Set up Prometheus config | `prometheus.yml` | [P] | - |
| T10-05 | Set up Grafana + Loki + Promtail docker-compose | `docker-compose.observability.yml` | [P] | - |
| T10-06 | Create Grafana dashboards (Business Overview + Technical Health) | Grafana JSON exports | - | T10-04, T10-05 |
| T10-07 | Configure Grafana alerts (error rate, server down, disk, latency) | Grafana alert rules | - | T10-06 |
| T10-08 | Implement rate limiting middleware | `internal/platform/middleware/ratelimit.go` | [P] | T0-07 |
| T10-09 | Write Dockerfile | `Dockerfile` | [P] | T0-01 |
| T10-10 | Write Nginx production config (SSL, reverse proxy, static, QR images) | Nginx config files | [P] | - |
| T10-11 | Set up systemd service | `turnero.service` | [P] | - |
| T10-12 | Set up GitHub Actions CI/CD (test + build + deploy) | `.github/workflows/deploy.yml` | [P] | - |
| T10-13 | Set up PostgreSQL backup cron | Backup script + crontab | [P] | - |
| T10-14 | Implement audit log repository + write on key actions | `internal/audit/repository.go` | [P] | T0-06 |

**Checkpoint:** Full observability stack running. CI/CD pipeline deploys on push to main. Backups automated.

---

## Phase 11: Web Mobile + Polish

| ID | Task | File(s) | Parallelizable | Dependencies |
|----|------|---------|:-:|---|
| T11-01 | Build Expo for Web and configure Nginx to serve | `turnero-app/`, Nginx config | - | T10-10 |
| T11-02 | Handle web-specific differences (manual phone input, no push, install banner) | `turnero-app/app/(client)/` components | - | T11-01 |
| T11-03 | Deep link handling (QR → app or web) | `turnero-app/app.json`, `turnero-app/app/_layout.tsx` | - | T4-06 |
| T11-04 | [Frontend] Landing/role selection screen | `turnero-app/app/index.tsx` | [P] | T0-11 |
| T11-05 | Client "recientes" (recently visited providers stored locally) | `turnero-app/stores/appStore.ts` | [P] | T0-13 |
| T11-06 | End-to-end testing of full flows | - | - | All phases |

**Checkpoint:** Web mobile works. Deep links route correctly. Full E2E flows verified.

---

## Dependency Graph (Phases)

```
Phase 0 (Setup)
    ├── Phase 1 (Auth)
    │       ├── Phase 2 (Services + Schedules)
    │       │       └── Phase 3 (Availability + Booking)
    │       │               ├── Phase 4 (QR + WhatsApp) ────────────────┐
    │       │               ├── Phase 5 (PJ + Employees) ──────────────┤
    │       │               │       └── Phase 6 (Cancel/Reschedule)    │
    │       │               ├── Phase 7 (Push Notifications)           │
    │       │               │       └── Phase 8 (Statistics)           │
    │       │               └── Phase 9 (Billing) ─────────────────────┤
    │       │                                                           │
    │       └── Phase 10 (Observability + DevOps) ─── independent ─────┤
    │                                                                   │
    └── Phase 11 (Web Mobile + Polish) ────────────────────────────────┘
```

**Parallel Opportunities Summary:**
- Phase 0: Backend setup (T0-01 to T0-10) || Frontend setup (T0-11 to T0-14)
- Phase 1: Backend auth (T1-01 to T1-08) || Frontend auth (T1-09 to T1-12) after API client
- Phase 3: Client booking UI || Provider agenda UI (T3-09 to T3-14 || T3-15 to T3-17)
- Phase 4: QR + WhatsApp — entirely parallelizable with Phase 5, 7, 8, 9
- Phase 5 + Phase 7: Can run in parallel (employees vs notifications)
- Phase 8 + Phase 9: Can run in parallel (stats vs billing)
- Phase 10: Observability is mostly independent, can run alongside any phase

---

## Estimated Effort (1 Developer)

| Phase | Estimated Time | Cumulative |
|-------|---------------|------------|
| Phase 0: Setup | 2-3 days | 3 days |
| Phase 1: Auth | 2-3 days | 6 days |
| Phase 2: Services + Schedules | 2-3 days | 9 days |
| Phase 3: Availability + Booking | 5-7 days | 16 days |
| Phase 4: QR + WhatsApp | 1-2 days | 18 days |
| Phase 5: PJ + Employees | 4-5 days | 23 days |
| Phase 6: Cancel/Reschedule | 2-3 days | 26 days |
| Phase 7: Push Notifications | 3-4 days | 30 days |
| Phase 8: Statistics | 1-2 days | 32 days |
| Phase 9: Billing | 3-4 days | 36 days |
| Phase 10: Observability + DevOps | 3-4 days | 40 days |
| Phase 11: Web Mobile + Polish | 2-3 days | 43 days |
| **Total** | **~8-10 weeks** | |

**MVP (Phases 0-4):** ~3-4 weeks — Provider registers, configures services/schedule, client books via QR/search, WhatsApp integration.
