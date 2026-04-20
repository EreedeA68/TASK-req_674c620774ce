# Keystone

A fullstack offline operations management system combining candidate intake, lost-and-found tracking, and automotive part master data for regulated organizations. Runs entirely on-premise with no internet connectivity required.

## Architecture & Tech Stack

- **Frontend**: React 18, TailwindCSS, React Router v6
- **Backend**: Go 1.22, Echo framework, GORM
- **Database**: PostgreSQL 15
- **Containerization**: Docker & Docker Compose

## Project Structure

```
keystone/
├── backend/
│   ├── cmd/server/main.go          # Entry point
│   ├── internal/
│   │   ├── auth/                   # Authentication & sessions
│   │   ├── candidate/              # Candidate intake workflow
│   │   ├── lostfound/              # Lost & found listings
│   │   ├── parts/                  # Automotive parts catalog
│   │   ├── search/                 # Cross-module search
│   │   ├── reports/                # KPI dashboards & exports
│   │   ├── audit/                  # Immutable audit log
│   │   ├── documents/              # Document management
│   │   ├── middleware/             # JWT & RBAC middleware
│   │   └── db/                     # GORM models & connection
│   └── pkg/
│       ├── crypto/                 # AES-256 encryption
│       ├── totp/                   # TOTP MFA (offline)
│       └── similarity/             # Levenshtein fuzzy match
├── frontend/
│   └── src/
│       ├── components/             # Shared UI components
│       ├── pages/                  # Route-level pages
│       ├── context/                # Auth context
│       ├── services/               # Axios API client
│       └── utils/                  # Date formatting helpers
├── db/
│   └── init.sql                    # Schema, indexes, seed data
├── tests/
│   ├── unit/                       # Go unit tests
│   ├── integration/                # Go httptest integration tests
│   ├── frontend/                   # Jest + RTL component tests
│   └── e2e/                        # Playwright end-to-end tests
├── docker-compose.yml
├── .env.example
├── run_tests.sh
└── README.md
```

## Prerequisites

- Docker >= 24.0
- Docker Compose >= 2.0

No other local dependencies required.

## Running the Application

```bash
# 1. Copy environment config
cp .env.example .env

# 2. Build and start all services
docker-compose up --build -d

# 3. Access the application
# Frontend:     http://localhost:3000
# Backend API:  http://localhost:8080/api
# Health check: http://localhost:8080/api/health
```

## Testing

```bash
chmod +x run_tests.sh
./run_tests.sh
```

The script will:
1. Build all Docker images
2. Start the test environment
3. Run backend unit & integration tests (Go)
4. Run frontend component tests (Jest + RTL)
5. Run E2E browser tests (Playwright)
6. Print coverage summary
7. Tear down and exit 0 only if all tests pass and coverage thresholds met (backend >= 80%, frontend >= 75%)

## Seeded Credentials

| Role              | Username            | Password           |
|-------------------|---------------------|--------------------|
| Admin             | admin               | Admin@Keystone1!   |
| Intake Specialist | intake_specialist   | Intake@Keystone1!  |
| Reviewer          | reviewer            | Review@Keystone1!  |
| Inventory Clerk   | inventory_clerk     | Clerk@Keystone1!   |
| Auditor           | auditor             | Audit@Keystone1!   |

## Security Features

- **Authentication**: Local username/password only (no OAuth, no SSO)
- **Password policy**: Minimum 12 characters, requires number and symbol
- **Account lockout**: 15-minute lockout after 5 failed attempts
- **MFA**: TOTP-based (RFC 6238), fully offline -- no SMS or email
- **Encryption at rest**: AES-256-GCM for all sensitive fields (MFA secrets)
- **Document integrity**: SHA-256 hash stored and verified on every download
- **Watermarking**: Optional on generated download copies
- **RBAC**: Role enforced on every endpoint down to menu visibility
- **Audit log**: Immutable, full-chain -- logins, CRUD, approvals, downloads

## Roles & Permissions

| Role              | Candidates | Listings | Parts | Audit | Admin |
|-------------------|-----------|---------|-------|-------|-------|
| ADMIN             | Full      | Full    | Full  | View  | Full  |
| INTAKE_SPECIALIST | Create/Edit| View   | --    | --    | --    |
| REVIEWER          | Approve   | Override| --    | --    | --    |
| INVENTORY_CLERK   | --        | Create  | Full  | --    | --    |
| AUDITOR           | Read-only | Read    | Read  | View  | --    |

## Offline Operation

Keystone is designed for fully air-gapped environments:
- No external API calls anywhere in the codebase
- No webhooks, no cloud storage, no maps, no payment processors
- TOTP MFA uses locally stored shared secrets (AES-256 encrypted)
- All documents stored on the local filesystem under `DOCUMENTS_PATH`
- PostgreSQL runs as a local container with persistent volume
