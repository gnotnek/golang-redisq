## 🚀 Go + Redis Job Queue — Full Explanation & Flow Diagram

---

### 1. Project Overview
a **distributed job queue system** using Go and Redis. The project demonstrates how to design background task processing with retries, scheduling, and fault tolerance.

- **Language**: Go
- **Storage/Queue**: Redis (Streams + Sorted Set)
- **Architecture**: Clean Architecture (domain, usecase, infra, cmd)
- **Key Features**:
  - Job enqueueing (API or producer)
  - Scheduled/delayed jobs
  - Worker with retry + exponential backoff
  - Dead-letter queue (DLQ)
  - Horizontal scalability with consumer groups

---

### 2. Flow Diagram

```
                ┌───────────────┐
                │   Producer    │
                │ (API/Client)  │
                └───────┬───────┘
                        │
                        │ enqueue job
                        ▼
                 ┌───────────────┐
                 │ Redis ZSET    │ ◄─── scheduled jobs
                 │ (delayed)     │
                 └───────┬───────┘
                         │ scheduler moves ready jobs
                         ▼
                 ┌───────────────┐
                 │ Redis Stream  │
                 │ (main queue)  │
                 └───────┬───────┘
                         │ consumer group
                         ▼
                ┌────────────────┐
                │    Worker      │
                │  (usecase)     │
                └───────┬────────┘
                        │
          ┌─────────────┴─────────────┐
          ▼                           ▼
   success → ACK                failure → Retry
                                        │
                                        ▼
                            Max attempts exceeded?
                                        │
                        ┌───────────────┴───────────────┐
                        ▼                               ▼
                  Retry (backoff)                  DLQ Stream
```

---

### 3. Detailed Data Flow

1. **Producer (API/Client)**
   - Calls `/enqueue` endpoint.
   - Job is added into:
     - **Redis Stream** (immediate execution), or
     - **Redis ZSET** (scheduled execution with timestamp score).

2. **Scheduler**
   - Runs in background inside the worker service.
   - Checks ZSET every second.
   - If jobs are due → moves them from ZSET → Stream.

3. **Worker (Consumer)**
   - Uses `XREADGROUP` from a Redis consumer group.
   - Each worker has a **ConsumerName**.
   - Reads jobs and passes them to handler.

4. **Handler Execution**
   - Executes the business logic.
   - Example: demo handler fails the first 2 attempts if type = `demo.fail`.

5. **Retry / DLQ Logic**
   - If handler fails → job is retried with exponential backoff.
   - Retries continue until `maxAttempts` reached.
   - If all retries fail → job is moved to **DLQ stream**.

6. **Observability (optional next steps)**
   - Metrics (e.g., Prometheus) can track retries, failures, DLQ counts.
   - Dead jobs can be reprocessed manually from DLQ.

---

### 4. Key Learnings

1. **Clean Architecture applied to distributed systems**
   - `domain`: business entities (Task).
   - `usecase`: application logic (Consumer, retry, scheduler).
   - `infra`: Redis implementation.
   - `cmd`: entrypoints with Cobra CLI.

2. **Redis as a message queue**
   - **Streams** provide ordered, persistent logs.
   - **Consumer groups** allow horizontal scaling of workers.
   - **ZSET** implements delayed jobs.
   - **DLQ** ensures errors don’t kill the system.

3. **Distributed system challenges**
   - Handling retries without spamming → exponential backoff.
   - Preventing lost jobs → Streams persistence + DLQ.
   - Ensuring idempotency → handlers must handle duplicate jobs.

4. **Practical DevOps lessons**
   - Config via environment variables (12-factor app).
   - Running local Redis + worker processes.
   - Using CLI (Cobra) to manage multiple commands.

---

### 5. Something that can be improve
- **Idempotency Keys** → ensure no duplicate processing.
- **Reaper (XAUTOCLAIM)** → detect stuck jobs.
- **Metrics Dashboard** → with Prometheus + Grafana.
- **Admin API** → inspect DLQ, requeue jobs.
- **Horizontal Scaling Demo** → run multiple workers to show load balancing.

---

✅ It's basically a **miniature version of Sidekiq / Celery / Faktory** in Go with Redis — but with clean architecture.

