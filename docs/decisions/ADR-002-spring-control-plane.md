# ADR-002: Use Spring Boot for the Control Plane

* **Status**: Accepted
* **Deciders**: Systems & Architecture Team
* **Date**: 2026-09-27

---

## 1. Context & Problem Statement
The Control Plane (`apps/radar-control-plane`) is responsible for managing query history, aggregating financial cost savings, computing hit-rate distributions across multiple threshold values, and persisting logs to relational databases (PostgreSQL / H2).

Unlike the proxy data plane (which requires microsecond CPU speed), the control plane requires:
1. Robust transactional persistence and schema migrations (Flyway).
2. Complex SQL aggregations (sums, averages, percentiles).
3. Enterprise-grade testing tools (Testcontainers, JUnit 5).
4. Long-term reliability and standard REST API contracts.

---

## 2. Options Considered

### Option 1: Spring Boot 3.3+ / Java 21 (Selected)
* **Pros**:
  - Industry-standard Spring Data JPA and Hibernate for relational persistence.
  - Native Flyway integration for versioned database migrations.
  - Clean separation of concerns (Controllers, Services, Repositories, Entities).
  - Production-ready metrics, health endpoints, and auditing.
  - Native Java 21 features (virtual threads, records, pattern matching).
* **Cons**:
  - Higher memory usage than Go (JVM footprint ~150–250MB).
  - Slower cold startup time (~2.5 seconds).

### Option 2: Go for Control Plane (Unified Monolith)
* **Pros**: Single language across backend, tiny binary, fast startup.
* **Cons**: Go's ORM ecosystem (GORM) is less suited for complex enterprise relational modeling; lacks the mature testing infrastructure and standardized Spring Boot ecosystem.

### Option 3: Python / FastAPI
* **Pros**: Great for quick ML scripts.
* **Cons**: Dynamic typing challenges at scale; Python relational ORMs (SQLAlchemy) have lower transactional throughput than Java HikariCP.

---

## 3. Decision
Use **Spring Boot 3.3+** with **Java 21** as the dedicated Control Plane service (`apps/radar-control-plane`).

---

## 4. Consequences & Trade-offs
* **Positive**:
  - Decoupled from proxy latency: The Go proxy flushes batches asynchronously to Spring Boot every 200ms via non-blocking HTTP POST.
  - Rich analytical endpoints (`/api/v1/analytics/summary`, `/api/v1/analytics/recent`, threshold sensitivity curves).
  - Seamless database portability (H2 in-memory for local dev, PostgreSQL for production Docker).
* **Negative**:
  - Requires Java 21 runtime. Mitigated by containerization and Gradle Wrapper (`gradlew`).
