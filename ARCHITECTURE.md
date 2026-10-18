# Application Architecture

Todo Planner is a personal sustainability planner for designing, executing, and
reviewing versioned ideal-day structures. The application separates two
different user moments:

- **Live surface**: a small, persistent interface for recording activity with
  minimal interruption during the day.
- **Review surface**: an intentionally opened interface for editing timelines,
  reviewing past days, and refining templates.

The web application is the review and account-management surface. The native
clients are live logging surfaces. All durable user data is owned by the Go API
and PostgreSQL; clients may cache or queue data locally, but they are not the
source of truth.

For backend layering, repository conventions, HTTP error handling, and Go file
organization, see [api-backend/ARCHITECTURE.md](./api-backend/ARCHITECTURE.md).
For endpoint and database details, see [docs/API and DB.md](./docs/API%20and%20DB.md).

## System Boundary

```text
                         +----------------------+
                         |   PostgreSQL         |
                         |   durable user data  |
                         +----------^-----------+
                                    |
                                    | pgx connection pool
                                    |
+-------------+      HTTPS/JSON     |
| front-web   | <----------------> +----------------------+
| review UI   |                     | api-backend          |
+-------------+                     | auth, validation,    |
                                    | domain workflows,   |
+-------------+      HTTPS/JSON     | event derivation,   |
| macOS       | <----------------> | persistence         |
| live widget |                     +----------------------+
+-------------+
                                    ^
+-------------+      future API    |
| RN mobile   | -------------------+
| live widget |   currently mock repository
+-------------+
```

The API is intentionally a thin synchronization and domain boundary rather
than a separate frontend-for-each-client backend. It authenticates requests,
enforces user ownership, applies date and timeline rules, derives actual blocks
from events, and persists the resulting state.

## Repository Layout

| Directory or file | Role |
| --- | --- |
| `api-backend/` | Go HTTP API, PostgreSQL repositories, services, migrations, and integration tests |
| `front-web/` | React/Vite review application and account/configuration UI |
| `front-desktop-widget/` | macOS SwiftUI/AppKit live widget with local event storage and remote sync |
| `front-rn-mobile/` | Expo React Native live-widget client; API integration is not yet the active repository implementation |
| `docs/` | Product requirements, API/database contract, and client design specifications |
| `docker-compose.yaml` | Local PostgreSQL and API orchestration, including the test database |
| `Makefile` | Repository-level development, formatting, linting, test, and local-stack commands |

Each client owns its presentation, transient UI state, and platform-specific
interaction model. Shared behavior is expressed through the HTTP contract and
the domain rules in the API, not through a shared code package.

## Client Responsibilities

### Web Review Client

`front-web` is a React 19 application built with Vite, TypeScript, Wouter,
Zustand, TanStack Query, and dnd-kit. It uses hash-based routing and stores the
JWT in the web client's auth store. API service modules translate UI actions
into requests; feature stores hold loaded data and coordinate refreshes.

The implemented routes cover:

- authentication and token transfer to the widget;
- categories and optional Pomodoro configuration;
- templates and their draggable timeline editor;
- weekly schedule and date overrides;
- weekly plan-versus-actual review and single-day editing;
- user preferences.

The web client is the place where users create and edit plans, correct actual
blocks, manage account settings, and initiate backup or native-client setup.
It should not reproduce server-side event derivation or make persistence
decisions based only on its local state.

### macOS Live Widget

`front-desktop-widget` is a SwiftUI/AppKit menu-bar application. It presents a
small always-available live surface with keyboard-first controls:

- confirm the planned activity;
- switch category with `1` through `9`;
- record notes;
- return to the planned category;
- nudge a delayed transition timestamp;
- manually synchronize with the API.

The widget uses `TodoPlannerRepository` as its client-side abstraction. The
remote implementation calls the API, while the local event store keeps native
events queued until a sync boundary. The widget performs bootstrap on startup,
can sync on explicit user action and lifecycle boundaries, and must retain
queued events until the API acknowledges them as accepted or duplicate.

Authentication starts in the web application. The web token page transfers a
JWT to the widget through the supported browser/deep-link flow; the widget
stores credentials using platform storage and does not provide a separate
account-registration flow.

### React Native Live Client

`front-rn-mobile` is an Expo Router application whose current implementation is
deliberately isolated around a mock repository. Its reducer and shell model the
same live states as the desktop widget: loading, confirmation prompt, active
activity, off-schedule activity, and no selected category.

When mobile networking is enabled, the mock repository should be replaced by a
repository implementing the same client boundary as the desktop client. The
mobile UI should remain a live surface rather than growing the web review
navigation into the native application.

## Request and Synchronization Flows

### Web Request Flow

```text
browser route/component
    -> feature service module
    -> JWT Authorization header
    -> API route and middleware
    -> handler validation
    -> service for workflows, or repository for simple CRUD
    -> PostgreSQL transaction/query
    -> JSON response
    -> client store refresh
```

The API applies logging, CORS, and authentication at the HTTP boundary. Public
routes are health and authentication endpoints. All user data routes require a
valid JWT and scope every read and write to the authenticated user.

### Native Bootstrap Flow

On first native-client setup, the client obtains a token through the web flow,
registers a device with `POST /devices`, and stores the returned `device_id`
locally. It then calls `POST /init` for settings, active categories, and the
requested day record. The bootstrap response is sufficient to render the live
surface without a sequence of dependent reads.

The device identifier distinguishes event sources. It is not an authorization
substitute: the JWT still identifies the user, and the API verifies that the
device belongs to that user.

### Native Event Sync Flow

```text
user interaction
    -> local event with client_event_id
    -> local queue and optimistic widget state
    -> POST /days/{date}/events (possibly as a batch)
    -> authenticate user and validate device ownership
    -> create day record if needed
    -> idempotently insert events
    -> apply amendments and derive actual blocks
    -> commit transaction
    -> accepted IDs and duplicate IDs returned
    -> remove only acknowledged events from local queue
```

Native clients may queue events while offline or while minimizing network
traffic. `client_event_id` makes retries safe. A client must not discard an
event merely because a request completed; it removes the event only after its
ID appears in `accepted_events` or `duplicate_client_event_ids`.

The desktop widget also supports local bootstrap and event persistence so the
live interaction remains usable when the API is unavailable. The server
remains responsible for canonical ordering, amendment validation, and actual
block derivation.

### Review Correction Flow

The web review surface can replace a day's actual block list with
`PUT /days/{date}/blocks`. This is a deliberate correction or reconstruction
path and is distinct from appending live events. The server validates alignment,
duration, overlap, category ownership, and blank-block rules, then derives
untracked gaps for the returned day representation.

Subsequent live event behavior must follow the API contract. Clients should
display the returned server state rather than attempting to merge independently
calculated timelines.

## Domain Model and Ownership

The core relationship is:

```text
User
  +-- User Settings
  +-- Devices
  +-- Activity Categories
  +-- Template Groups
  +-- Day Templates
  |     +-- Template Snapshots
  |           +-- Snapshot Blocks
  +-- Weekly Schedule / Schedule Overrides
  +-- Day Records
        +-- Day Events
        +-- Actual Blocks
```

- **Category** is a user-defined classification with name, color, and optional
  Pomodoro settings. Soft deletion hides it from new assignments but preserves
  historical meaning.
- **Day Template** is a named ideal-day structure. Each edit creates a retained
  snapshot containing its planned blocks.
- **Schedule** resolves a calendar date to a template, with a date override
  taking precedence over the weekly assignment.
- **Day Record** is the date-keyed container for a planned snapshot, raw events,
  timezone-offset metadata, and actual blocks.
- **Day Event** is the append-oriented input from live tracking. Transitions,
  confirmations, notes, and amendments are retained for audit and replay.
- **Actual Block** is the materialized timeline used by review and analysis. It
  may be derived from events or replaced by a review correction.
- **Untracked** is a server-derived gap, not a stored user category. **Blank**
  is an explicit review block marking time the user cannot recall; the two must
  remain visually and analytically distinct.

## Time and Versioning Rules

The system uses three separate time representations:

- date-only values use `YYYY-MM-DD`;
- schedule and block times use `HH:MM:SS` within a calendar day;
- server timestamps use UTC RFC3339 values ending in `Z`;
- client local event timestamps preserve the original numeric UTC offset.

The literal `calendar_date` in the route and event payload determines the day
record. The tracking-day range is only a rendering hint for the web and native
surfaces; it does not move events across dates or change derivation.

The API pins a template snapshot to a day record. Past records are frozen to
their pinned snapshot when templates or schedules change. Active and future
records may be re-resolved and re-pinned, while their events and actual blocks
remain unchanged. This preserves historical comparisons between intended and
lived structure.

When events are submitted, the API applies amendments, sorts by effective time
with a deterministic server tie-breaker, rejects invalid non-monotonic
corrections, and recomputes actual blocks transactionally. Clients should treat
the response as authoritative.

## Security and Data Isolation

- Registration and login issue JWTs; protected requests send the token through
  the `Authorization` header.
- Passwords are stored as hashes. Tokens, passwords, password hashes, and
  authorization headers must never be written to logs or responses.
- Every repository query for user-owned data includes the authenticated
  `user_id`. IDs supplied by a client never authorize access to another user's
  records.
- Device ownership is checked against the authenticated user.
- Resource deletion is soft by default. Full account deletion is the only hard
  delete and removes all user-owned data in the server workflow.
- CORS origins, database connection details, JWT secret, and server port are
  supplied through environment configuration rather than source code.

## Runtime and Deployment

The backend supports two equal runtime modes selected with `APP_MODE`:

- `cloud` uses the required external `DATABASE_URL` and is the deployment mode
  used by Docker/Railway.
- `local` starts embedded PostgreSQL, persists it under
  `EMBEDDED_PG_DATA_DIR`, and binds the API to loopback for the desktop app.

For cloud-mode local development, `docker-compose.yaml` runs PostgreSQL and the
API. The web client runs through Vite on the host, normally at
`http://localhost:5173`, and calls the API at `http://localhost:3648` unless
`VITE_API_URL` overrides it. The API applies migrations at startup and uses a
deliberately small, shared `pgxpool` connection pool for the process lifetime.

The API is packaged by `api-backend/Dockerfile`. The root `railway` Make target
deploys that backend directory as the service root. Native clients are built
independently: Expo commands build the mobile client and the widget Makefile
builds and tests the macOS application.

## Testing and Change Boundaries

Tests are organized by owning surface:

- backend repository tests run against PostgreSQL; handler and domain tests use
  focused HTTP or isolated dependencies;
- web tests cover service modules, Zustand stores, timeline utilities, and
  serialization behavior;
- widget tests cover state transitions, persistence, authentication, deep links,
  and repository synchronization;
- mobile tests target the reducer and live-widget behavior as the client grows.

Use repository-level commands for normal verification:

```text
make lint
make format
make test
make test-local
make build-local
```

`make test` preserves the Docker-backed cloud database test workflow. The
additional `make test-local` target runs the same Go tests against a temporary
embedded PostgreSQL instance.

Make a feature change in the owning client or backend layer and update the
contract documentation when behavior changes. Do not add client-specific API
workarounds for a server rule that belongs in the API. Do not move domain
derivation into a UI merely to make a screen render faster; optimize only after
preserving the server as the source of truth.

## Adding a Cross-Client Feature

1. Define the user-visible behavior and persistence contract in `docs/`.
2. Update the PostgreSQL migration and backend repository/service/handler flow.
3. Add focused backend tests for authorization, validation, state transitions,
   idempotency, and error paths.
4. Update the web service module, store, and review/configuration UI.
5. Update the native repository models and local persistence if the feature is
   part of the live flow.
6. Verify the API contract from each supported client and document any platform
   limitation instead of silently diverging behavior.

Keep the API contract stable across clients, keep presentation platform-specific,
and preserve the distinction between planned structure, recorded events, actual
blocks, blank time, and untracked time.
