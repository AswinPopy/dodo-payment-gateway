Payment Service Design & Architecture
--------------------------------------------

The system consists of two primary services:
Merchant → Invoice API (:8080) → Mock PSP (:8081)
                    ↓
                PostgreSQL
The Invoice API manages businesses, customers, invoices, and payments. The Mock PSP operates as an isolated HTTP service so that real-world failure modes—such as timeouts and network drops—behave identically to a live third-party dependency.
The core engineering objective of this design is ensuring strict payment correctness: avoiding double charges under high concurrency, retries, and network partitions.

Data Model
-----------
The database schema is organized hierarchically per business:
businesses
    ├── customers
    ├── invoices
    │     └── payment_attempts
    ├── api_keys
    ├── idempotency_keys
    └── outbound_events

webhook_events
schema_migrations

Key Entities

    Businesses: Represents a merchant account onboarded to the gateway. Holds merchant        configuration, API access settings, and global webhook endpoints.

    Customers: Belongs to a business. Customer emails are scoped uniquely per business (UNIQUE(business_id, email)) to allow the same end-user email across distinct merchants.
    
    Invoices: Scoped to a business and customer. All monetary fields (amount) are stored as BIGINT (representing smallest currency units like cents) to avoid floating-point rounding errors. Line items are omitted for this specification.
    
    Payment Attempts: Tracks every charge attempt against an invoice. Monetary details are snapshot-copied from the invoice into the payment attempt record at creation, ensuring historical financial accuracy even if the parent invoice changes later.
    To enforce concurrency limits at the storage layer, a partial unique index is applied:
    CREATE UNIQUE INDEX idx_unique_pending_attempt 
    ON payment_attempts (invoice_id) 
    WHERE status = 'PENDING';
    
    Idempotency Keys: Keys are scoped to a merchant (UNIQUE(business_id, idempotency_key)). Each record stores a SHA-256 payload hash calculated from critical request fields: SHA256(invoice_id + card_token). Matching keys with identical payloads return the cached attempt result. Matching keys with mismatching payloads return an HTTP 409 Conflict.
    API Keys: Stored securely using SHA-256 hashing (SHA-256(full_key)). A plain-text prefix (e.g., dodo_live_...) is retained separately for log identification and revocation tracking.
    
    Webhook Events (Inbound): Logs incoming PSP webhooks. Dedupes callbacks using a unique constraint on the PSP event_id.
    
    Outbound Events: Implements the Transactional Outbox pattern for merchant notification hooks. Events are written to Postgres within the core payment transaction before delivery is attempted.

Identifiers & Database Mechanics
---------------------------------

Primary keys across all tables are application-generated UUIDs (preferring UUIDv7 for index B-tree locality). Utilizing application-side IDs allows log correlation and dependency-graph creation prior to committing database writes.
Invoice State Machine
Invoices follow a strict lifecycle transition model:
DRAFT → OPEN → PAID
           │
           ├──→ VOID
           │
           └──→ UNCOLLECTIBLE
PAID, VOID, and UNCOLLECTIBLE represent terminal states.

State Transitions
-------------------

Transition              Trigger Event
 
→ DRAFT                 Invoice entity initialized.
DRAFT → OPEN            First payment attempt initiated.
DRAFT/OPEN → PAID       PSP returns explicit charge success.
OPEN → VOID             Manual cancellation (reserved for extension).
OPEN → UNCOLLECTIBLE    Final write-off state (reserved for extension).

A declined payment attempt does not transition an invoice to a failed terminal state. An invoice remains OPEN while its attempt status transitions to FAILED, allowing subsequent retry attempts until payment settles.
Payment attempts use a simplified linear state flow: PENDING → SUCCEEDED or PENDING → FAILED. Terminal attempt states are immutable.

Payment Correctness & Concurrency Control
-----------------------------------------

To guarantee single-execution semantics during concurrent payment requests, the system coordinates database row locks, unique indexes, and upstream request hashes.
Incoming Request → Lock Invoice Row (FOR UPDATE)
                          │
                          ▼
             Check Existing Pending Attempt?
                   ├── Yes → Abort / Wait
                   └── No  → Insert PENDING Attempt → Dispatch to PSP

Concurrent Execution Flow
-------------------------
    Row Locking: Request A acquires an explicit lock on the target invoice via SELECT ... FOR UPDATE. Request B blocks waiting for the transaction lock to release.

    State Validation: Request A verifies the invoice is OPEN, creates a PENDING payment attempt record (enforced by the unique partial index), and releases its lock prior to making the network call to the PSP.

    Lock Resolution: Request B acquires the lock post-release, detects the existing PENDING attempt or updated PAID status, and exits cleanly without dispatching a secondary PSP charge.

Edge Case Mechanics
-------------------

    PSP Timeouts: If the upstream PSP fails to respond within the gateway's HTTP timeout threshold (e.g., 2 seconds vs. a delayed 30-second PSP processing delay), the payment attempt status remains PENDING and the invoice remains OPEN. Because network timeouts do not confirm failure, the state remains open until background reconciliation or inbound webhook callbacks resolve the attempt.
    
    Crashing Post-Charge: Before calling the PSP, the gateway persists the PENDING payment attempt ID. When issuing the PSP charge request, this attempt ID is sent as the PSP's external idempotency key. If the local service crashes mid-request, subsequent retries resend the same payment attempt ID, causing the PSP to return the original transaction result rather than creating a duplicate charge.
    
    Idempotency Misuse: Sending a previously used idempotency key alongside altered payload parameters (such as changing the target invoice_id) results in a payload hash mismatch, triggering an immediate HTTP 409 Conflict.
    
    Double Payment Prevention: Requests targeting an already PAID invoice return an immediate HTTP 409 Conflict, unless the request carries the original idempotency key that produced the successful payment—in which case the original successful record is returned.

Webhook Architecture
--------------------

Webhooks operate on a dual-direction model handling incoming provider notifications and outgoing merchant notifications independently.

Inbound (PSP → Invoice API)

Incoming callbacks (POST /webhooks/psp) are validated via HMAC-SHA256 signatures over a standardized payload digest:
  Signature = HMAC-SHA256(Secret, event_id + "." + timestamp + "." + raw_body)

Validation guards against replay attacks by rejecting incoming webhook headers where the Webhook-Timestamp diverges by more than 5 minutes from system time. Duplicate event deliveries are ignored at the database layer via unique event_id constraints.

Outbound (Invoice API → Merchant)

Merchant webhooks avoid synchronous network calls during the primary payment request flow.
Payment Request Success
          │
          ▼
DB Transaction (Atomic Write)
  ├── Payment Status  = SUCCEEDED
  ├── Invoice Status  = PAID
  └── Outbound Event  = PENDING
          │
          ▼
Return HTTP Response to Client
          │
          ▼
Asynchronous Background Worker → Deliver to Merchant Endpoint

Events exhausting all retries remain stored with their full error trace for manual operational replay.

Security Model
--------------

Authentication uses bearer tokens generated from cryptographically secure random bytes: dodo_live_8f91a...

Client Header → Bearer dodo_live_...
                         │
                         ▼
             SHA-256 Hash Computation
                         │
                         ▼
             Database Lookup (hashed_key)

Key features include:

    Storage Safety: Only the SHA-256 digest of an API key is stored. Raw key secrets are rendered once during initial generation or rotation.

    Immediate Revocation: Key authorization sets revoked_at = NOW(), causing all subsequent authorization requests to fail instantly.

    Zero-Downtime Rotation: Key updates generate a replacement secret before deprecating the previous identifier, avoiding merchant downtime during credential updates.

    Tenant Isolation: Keys are bound directly to business_id scopes, preventing cross-tenant access in multi-merchant environments.

Scope & Out-of-Scope Items
-------------------------

To keep the architecture focused on non-double-charge guarantees, specific domain features were excluded from the baseline implementation:

    Refunds & Adjustments: Database states account for void/uncollectible flows, but external API surfaces for partial/full refunds or credit notes are omitted.
    
    Mock PSP Persistence: The mock PSP retains idempotency states in ephemeral memory. In a live production environment, the PSP requires persistent storage to guarantee idempotency across provider node restarts.

    Granular Key Permissions: API keys currently carry full administrative access for the underlying business account rather than fine-grained scope limitations (e.g., invoices:read).
    
    Ledger Mechanics: Financial balances are calculated off invoice states rather than an immutable double-entry ledger system.

Production Roadmap
------------------

Priorities for scaling the service to production infrastructure include:
    Observability & Alerting:
        Metrics for payment attempt latencies, PSP timeout ratios, and webhook queue backlogs.
        Automated alerts targeting PENDING payment records exceeding 15 minutes, signaling unresolved PSP states requiring automated reconciliation.

    Reconciliation Engine:
        A scheduled batch process to sync stagnant PENDING transactions against provider ledger logs to settle orphaned states caused by upstream network drops.

    Granular Access & Rate Limiting:
        Token scoping limits (e.g., separating read keys from payment submission keys) paired with token-bucket rate limiters applied at the HTTP gateway layer.
