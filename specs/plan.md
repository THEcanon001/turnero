# Implementation Plan: Turnero App

---

## Summary

Turnero App is a generic appointment scheduling platform for LatAm service providers. The implementation follows a phased approach aligned with the user story priorities (P1-P4). The backend is a monolithic Go HTTP server with PostgreSQL, and the frontend is a single Expo codebase for iOS, Android, and Web.

---

## Technical Context

| Layer | Technology | Version |
|-------|------------|---------|
| Backend language | Go | 1.22+ |
| HTTP router | Chi (go-chi/chi/v5) | v5 |
| Database | PostgreSQL | 16 |
| DB driver | pgx/v5 + pgxpool | v5 |
| Migrations | golang-migrate/migrate/v4 | v4 |
| Auth | JWT (golang-jwt/jwt/v5) + bcrypt | v5 |
| Validation | go-playground/validator/v10 | v10 |
| QR generation | skip2/go-qrcode | - |
| Push notifications | Firebase Admin SDK (FCM) | v4 |
| Cron jobs | robfig/cron/v3 | v3 |
| Config | caarlos0/env/v11 | v11 |
| Logging | log/slog (stdlib) | stdlib |
| Frontend framework | Expo (React Native + Web) | SDK 52+ |
| Navigation | expo-router | v4 |
| Styling | NativeWind (Tailwind for RN) | - |
| State management | Zustand + TanStack Query | - |
| Forms | react-hook-form + zod | - |
| Camera/QR | expo-camera | - |
| Push (client) | expo-notifications | - |
| Hosting | Hetzner Cloud VPS (CX22/CX32) | - |
| Reverse proxy | Nginx | - |
| SSL | Let's Encrypt (Certbot) | - |
| Observability | Prometheus + Grafana + Loki | self-hosted |
| CI/CD | GitHub Actions | - |

---

## Project Structure

### Go Backend — Package Oriented Design (POD)

Packages are organized **by domain**, not by technical layer. Each domain package owns its types, handlers, services, and repositories. Cross-cutting infrastructure lives in `internal/platform/`.

```
turnero/
├── cmd/
│   └── server/
│       └── main.go                        # Entry point, wiring, graceful shutdown
├── internal/
│   ├── platform/                          # Cross-cutting infrastructure
│   │   ├── config/
│   │   │   └── config.go                  # Env vars → struct (caarlos0/env)
│   │   ├── database/
│   │   │   ├── database.go                # pgxpool connection setup
│   │   │   └── migrations/
│   │   │       ├── 001_initial.up.sql     # Full schema (see data-model.md)
│   │   │       ├── 001_initial.down.sql
│   │   │       └── ...
│   │   └── middleware/
│   │       ├── auth.go                    # JWT validation, extract claims
│   │       ├── cors.go                    # CORS (go-chi/cors)
│   │       ├── ratelimit.go              # In-memory token bucket (x/time/rate)
│   │       ├── logging.go                 # slog request logging
│   │       ├── metrics.go                 # Prometheus HTTP metrics
│   │       └── recovery.go               # Panic recovery
│   ├── auth/                              # Domain: authentication
│   │   ├── model.go                       # LoginRequest, RegisterRequest, TokenPair
│   │   ├── handler.go                     # POST /auth/register, login, refresh, logout
│   │   ├── service.go                     # bcrypt hash/verify, JWT sign
│   │   └── repository.go                 # refresh_tokens queries
│   ├── provider/                          # Domain: provider profile
│   │   ├── model.go                       # Provider, ProviderType structs + validation
│   │   ├── handler.go                     # GET/PUT /provider/me, settings
│   │   └── repository.go                 # SQL queries for providers
│   ├── employee/                          # Domain: employees + invitations
│   │   ├── model.go                       # Employee, InvitationCode structs
│   │   ├── handler.go                     # CRUD /employees, invitations, join
│   │   ├── service.go                     # Invitation logic, deactivation
│   │   └── repository.go                 # SQL queries for employees + employee_services
│   ├── schedule/                          # Domain: schedules + exceptions
│   │   ├── model.go                       # Schedule, ScheduleException structs
│   │   ├── handler.go                     # CRUD /employees/:id/schedules, exceptions
│   │   └── repository.go                 # SQL queries for schedules
│   ├── appointment/                       # Domain: appointments + availability
│   │   ├── model.go                       # Appointment, AppointmentStatus structs
│   │   ├── handler.go                     # Create, cancel, status, list, reschedule
│   │   ├── service.go                     # Slot calculation, conflict detection
│   │   └── repository.go                 # SQL queries for appointments
│   ├── billing/                           # Domain: billing + monetization
│   │   ├── model.go                       # BillingUsage, BillingTransaction structs
│   │   ├── handler.go                     # GET /provider/me/billing
│   │   ├── service.go                     # Monthly usage calc, cap logic, grace period
│   │   ├── repository.go                 # SQL queries for billing_usage + transactions
│   │   └── worker.go                      # Cron: monthly billing cycle
│   ├── notification/                      # Domain: push notifications
│   │   ├── model.go                       # PushToken struct
│   │   ├── handler.go                     # POST/DELETE /push-tokens
│   │   ├── service.go                     # FCM push sender
│   │   ├── repository.go                 # SQL queries for push_tokens
│   │   └── worker.go                      # Cron: reminders, weekly summary
│   ├── search/                            # Domain: search
│   │   └── handler.go                     # GET /search?q=
│   ├── qr/                                # Domain: QR generation
│   │   ├── handler.go                     # GET /provider/me/qr
│   │   └── service.go                     # QR image generation (go-qrcode)
│   ├── audit/                             # Domain: audit logging
│   │   └── repository.go                 # SQL queries for audit_log
│   └── worker/                            # Shared worker setup
│       └── metrics.go                     # Cron: update Prometheus business gauges
├── pkg/
│   └── jwt/
│       └── jwt.go                         # JWT helper (generate/validate/claims)
├── static/                                # Expo Web build output (served by Nginx)
├── go.mod
├── go.sum
├── Makefile                               # build, test, migrate, run targets
├── Dockerfile
└── docker-compose.yml                     # Dev: Go + PostgreSQL
```

### Expo Frontend

```
turnero-app/
├── app/                              # expo-router (file-based routing)
│   ├── _layout.tsx                   # Root layout (providers, auth check)
│   ├── index.tsx                     # Landing / role selection
│   ├── (auth)/
│   │   ├── login.tsx                 # Provider login
│   │   └── register.tsx              # Provider registration
│   ├── (client)/
│   │   ├── _layout.tsx              # Client tab layout
│   │   ├── index.tsx                # Client home (recientes, buscar)
│   │   ├── search.tsx               # Search providers by name/slug
│   │   ├── provider/
│   │   │   └── [slug].tsx           # Provider profile + services
│   │   ├── book/
│   │   │   └── [slug].tsx           # Booking flow (service → employee → date → time → confirm)
│   │   ├── appointments/
│   │   │   ├── index.tsx            # "Mis turnos" list
│   │   │   └── [id].tsx            # Appointment detail (cancel, reschedule, WhatsApp)
│   │   └── scan.tsx                 # QR scanner (expo-camera)
│   ├── (provider)/
│   │   ├── _layout.tsx              # Provider tab layout
│   │   ├── index.tsx                # Dashboard / agenda del día
│   │   ├── agenda/
│   │   │   ├── index.tsx            # Day/week view
│   │   │   └── [id].tsx            # Appointment detail (complete, no-show, cancel, WhatsApp)
│   │   ├── services/
│   │   │   ├── index.tsx            # List services
│   │   │   └── [id].tsx            # Edit service
│   │   ├── schedule/
│   │   │   ├── index.tsx            # Weekly schedule config
│   │   │   └── exceptions.tsx       # Block days / special hours
│   │   ├── employees/
│   │   │   ├── index.tsx            # Employee list (PJ only)
│   │   │   ├── [id].tsx            # Employee detail / assign services
│   │   │   └── invite.tsx           # Generate invitation code
│   │   ├── stats.tsx                # Statistics page
│   │   ├── billing.tsx              # Billing / usage page
│   │   ├── qr.tsx                   # QR generation + share
│   │   └── settings.tsx             # Rules, WhatsApp, notifications config
│   └── (employee)/
│       ├── _layout.tsx              # Employee tab layout
│       ├── index.tsx                # Employee agenda (own appointments only)
│       ├── agenda/
│       │   └── [id].tsx            # Appointment detail
│       ├── schedule.tsx             # Own availability config
│       └── join.tsx                 # Accept invitation code
├── components/
│   ├── ui/                          # Reusable UI primitives
│   │   ├── Button.tsx
│   │   ├── Card.tsx
│   │   ├── Input.tsx
│   │   ├── Modal.tsx
│   │   └── ...
│   ├── calendar/
│   │   ├── DayPicker.tsx            # Date selection calendar
│   │   ├── SlotGrid.tsx             # Available time slots
│   │   └── AgendaView.tsx           # Provider day/week agenda
│   ├── appointment/
│   │   ├── AppointmentCard.tsx      # Appointment summary card
│   │   └── StatusBadge.tsx          # Status indicator
│   └── provider/
│       ├── ProviderCard.tsx          # Provider search result card
│       └── ServiceList.tsx          # Service selection list
├── lib/
│   ├── api.ts                       # API client (fetch wrapper, auth headers)
│   ├── auth.ts                      # Auth context, token storage (SecureStore)
│   ├── notifications.ts             # FCM token registration
│   └── whatsapp.ts                  # WhatsApp deep link helper
├── hooks/
│   ├── useAuth.ts                   # Auth state hook
│   ├── useAppointments.ts           # TanStack Query: appointments
│   ├── useSlots.ts                  # TanStack Query: available slots
│   └── useProvider.ts               # TanStack Query: provider data
├── stores/
│   └── appStore.ts                  # Zustand: local state (role, recent providers)
├── constants/
│   └── config.ts                    # API URL, defaults
├── app.json                         # Expo config
├── package.json
└── tsconfig.json
```

---

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Code organization | Package Oriented Design (POD) | Packages organized by domain (auth, appointment, billing...) not by layer (handler, service, repository). Each domain owns its types, handlers, services, and repositories. Cross-cutting infra in `internal/platform/`. |
| Error handling | Layer-prefixed wrapping | Each layer prefixes errors: `fmt.Errorf("repository: %w", err)`. Upstream chain reads `handler: service: repository: actual error`. See **Error Handling Convention** below. |
| Monolith vs microservices | Monolith | Single dev, single VPS. Microservices add unnecessary complexity. Split later if needed. |
| ORM vs raw SQL | Raw SQL (pgx) | Full control over queries, better performance, no ORM magic. Model is simple enough. |
| Session vs JWT | JWT (stateless) | No server-side session storage. Access token 15min + refresh token 7d with rotation. |
| Client auth | None (phone-based identity) | Clients don't register. Phone is captured automatically. Cancel/verify by matching phone. |
| Push notification service | FCM (Firebase Cloud Messaging) | Free, cross-platform, reliable. No need for OneSignal/custom. |
| WhatsApp integration | Deep links (wa.me) | Zero cost, no API needed. Sufficient for MVP — upgrade to WhatsApp Business API later. |
| Cache | In-memory (sync.Map / LRU) | Single server, no need for Redis. Go handles in-memory cache trivially. |
| Message queue | None (goroutines) | Background workers use goroutines + channels. No external queue needed for MVP volume. |
| Rate limiting | In-memory (x/time/rate) | Single server. Move to Redis-backed when horizontally scaling. |
| Observability | Prometheus + Grafana + Loki | 100% free self-hosted. Professional-grade metrics and alerting. |
| Concurrency control (bookings) | Optimistic locking (DB UNIQUE constraint) | First-write-wins via UNIQUE(employee_id, date, start_time). No pessimistic locks that could block abandoned flows. |
| File storage (QR) | Local filesystem (VPS) | Served directly by Nginx. QR images are regenerable. Move to S3-compatible when scaling. |

---

## Error Handling Convention

Errors are wrapped at each layer with a prefix, creating a traceable chain upstream:

```
handler: service: repository: actual error
```

### Rules

1. **Originating errors** use `errors.New` with the current layer prefix:
   ```go
   // In repository
   return errors.New("repository: provider not found")
   ```

2. **Wrapping errors** from a lower layer use `fmt.Errorf` with `%w`:
   ```go
   // In service, wrapping a repository error
   if err != nil {
       return fmt.Errorf("service: %w", err)
   }
   // Result: "service: repository: provider not found"
   ```

3. **Handlers** wrap service errors before responding:
   ```go
   // In handler, wrapping a service error
   if err != nil {
       slog.Error("handler: " + err.Error())
       // Return appropriate HTTP status based on error type
   }
   // Log output: "handler: service: repository: provider not found"
   ```

4. **Use specific prefixes** matching the domain package name:
   - `auth:`, `provider:`, `appointment:`, `schedule:`, `billing:`, `notification:`, `employee:`, `qr:`
   - Within a domain, sub-prefix by layer: `appointment.repository:`, `appointment.service:`

### Example chain

```go
// internal/appointment/repository.go
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Appointment, error) {
    // ...
    if err != nil {
        return Appointment{}, fmt.Errorf("appointment.repository: %w", err)
    }
}

// internal/appointment/service.go
func (s *Service) Cancel(ctx context.Context, id uuid.UUID) error {
    apt, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return fmt.Errorf("appointment.service: %w", err)
    }
    // ...
}

// internal/appointment/handler.go
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
    err := h.service.Cancel(ctx, id)
    if err != nil {
        // Log: "appointment.handler: appointment.service: appointment.repository: connection refused"
        slog.Error(fmt.Sprintf("appointment.handler: %v", err))
    }
}
```

---

## Data Model

See [data-model.md](./data-model.md) for the complete SQL schema.

**Key tables:** `providers`, `employees`, `schedules`, `schedule_exceptions`, `appointments`, `billing_usage`, `billing_transactions`, `push_tokens`, `refresh_tokens`, `audit_log`.

---

## API Contracts

See [contracts/api.md](./contracts/api.md) for the complete REST API specification.

**Key endpoint groups:**
- Auth (public): register, login, refresh, logout
- Provider (authenticated): profile, QR, billing
- Employees (admin): CRUD
- Schedules (admin): weekly schedules, exceptions
- Availability (public): provider profile, employee slots
- Appointments (public for create/cancel, authenticated for manage)
- Search (public): by name/slug
- Push tokens (public): register/deregister FCM tokens

---

## Deployment Strategy

### MVP: SSH + systemd

```
Dev machine → GOOS=linux go build → scp → systemctl restart
```

- Go binary: single file, ~10-20MB
- PostgreSQL on same VPS
- Nginx reverse proxy with Let's Encrypt SSL
- Expo Web build served as static files by Nginx

### CI/CD: GitHub Actions

```yaml
on push to main:
  1. go test ./...
  2. GOOS=linux go build
  3. scp binary to VPS
  4. ssh: systemctl stop → mv binary → systemctl start
```

### Hosting

| Phase | Infrastructure | Cost/month |
|-------|---------------|------------|
| MVP (0-1K users) | Hetzner CX22 (2 vCPU, 4GB RAM) — everything on one VPS | ~$5-10 |
| Growth (1K-10K) | App VPS + Managed PostgreSQL | ~$20-35 |
| Scale (10K-100K) | Multiple Go instances + LB + Redis + Managed DB | ~$100-200 |

### Backups

- PostgreSQL: `pg_dump` daily at 3AM, retain 30 days
- QR images: rsync weekly (regenerable)
- Config: versioned in Git

---

## Observability

Self-hosted stack (Prometheus + Grafana + Loki) provides:

- **Technical metrics**: HTTP req/s, latency p50/p95/p99, error rate, goroutines, GC
- **Business metrics**: active providers, MRR, turnos completados, conversion rate, retention
- **System metrics**: CPU, RAM, disk (via node_exporter)
- **Database metrics**: queries/s, connections, slow queries (via postgres_exporter)
- **Logs**: structured slog → Promtail → Loki → Grafana
- **Alerts**: Grafana Alerting → Telegram/Email (error rate >5%, server down, disk >85%, latency p95 >500ms)

---

## Security

| Aspect | Implementation |
|--------|---------------|
| Auth | bcrypt (cost 12) + JWT (access 15min, refresh 7d with rotation) |
| SQL injection | Parameterized queries (pgx) |
| Input validation | go-playground/validator on all handlers |
| HTTPS | Let's Encrypt + Nginx, forced redirect |
| CORS | Whitelisted origins only |
| Rate limiting | Per-IP and per-user, in-memory |
| Client phone storage | Plain text (needed for WhatsApp). Consider AES-256-GCM encryption at app level. |
| Data retention | Auto-delete client data after 12 months of inactivity |
| Security headers | X-Content-Type-Options, X-Frame-Options, HSTS via Nginx |
| Dependency audit | go vet + govulncheck in CI |
