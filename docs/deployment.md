# Deployment & Infrastructure Guide

PromptRadar is designed to deploy seamlessly via Docker Compose for local evaluation or container orchestrators (Kubernetes / ECS) for production workloads.

```
                         Client Browser
                               │
            ┌──────────────────┼──────────────────┐
            │                  │                  │
        Port 4200          Port 8081          Port 8080
            ▼                  ▼                  ▼
     ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
     │   radar-ui   │   │  radar-proxy │   │control-plane │
     │ (Nginx 1.25) │   │ (Go Alpine)  │   │  (Java 21)   │
     └──────────────┘   └───────┬──────┘   └──────┬───────┘
                                │                 │
                       Async Batch Telemetry      │
                                └────────►────────┘
                                                  │
                                              Port 5432
                                                  ▼
                                           ┌──────────────┐
                                           │  PostgreSQL  │
                                           │  (pg 16-alp) │
                                           └──────────────┘
```

---

## 1. Quickstart: Docker Compose

From the root of the repository:

```bash
docker compose -f docker/docker-compose.yml up --build -d
```

To tail logs across all containers:
```bash
docker compose -f docker/docker-compose.yml logs -f
```

To shut down and wipe persistent volumes:
```bash
docker compose -f docker/docker-compose.yml down -v
```

---

## 2. Container Architecture

### 2.1 Angular Frontend (`prompt-radar-ui`)
* **Base Image**: Multi-stage build with `node:20-alpine` (builder) and `nginx:alpine` (production runtime).
* **Port**: Host `4200` $\to$ Container `80`.
* **Role**: Serves the compiled Angular 17 Single Page Application and proxies API requests.

### 2.2 Go Data Plane Proxy (`prompt-radar-proxy`)
* **Base Image**: Multi-stage build with `golang:1.22-alpine` (builder) and minimal `alpine:3.19` (runtime binary ~18MB).
* **Port**: Host `8081` $\to$ Container `8081`.
* **Role**: Ingests prompts, computes 768-D embeddings, evaluates vector similarity, serves 2ms cache hits, streams LLM misses via SSE, and queues async telemetry.

### 2.3 Spring Boot Control Plane (`prompt-radar-control-plane`)
* **Base Image**: Multi-stage build with `gradle:8.7-jdk21` (builder) and `eclipse-temurin:21-jre-alpine` (runtime).
* **Port**: Host `8080` $\to$ Container `8080`.
* **Role**: Ingests batch telemetry logs, executes Flyway migrations, and provides analytical aggregates.

### 2.4 PostgreSQL Database (`prompt-radar-postgres`)
* **Base Image**: `postgres:16-alpine`.
* **Port**: Host `5432` $\to$ Container `5432`.
* **Volume**: `postgres_data` mounted to `/var/lib/postgresql/data`.
* **Role**: Persistent ACID storage for historical telemetry logs and cost savings metrics.

---

## 3. Configuration & Environment Variables

| Service | Variable Name | Default | Required? | Description |
|---|---|---|---|---|
| `radar-proxy` | `PORT` | `8081` | No | Listening HTTP port. |
| `radar-proxy` | `CONTROL_PLANE_URL` | `http://radar-control-plane:8080/api/v1/telemetry` | Yes | Target endpoint for batch telemetry flush. |
| `radar-proxy` | `GEMINI_API_KEY` | *(None / Fallback)* | No | Optional Google AI Studio key for live Gemini 1.5 Flash completions. |
| `radar-control-plane` | `SPRING_DATASOURCE_URL` | `jdbc:postgresql://postgres:5432/prompt_radar` | Yes | JDBC connection string. |
| `radar-control-plane` | `SPRING_DATASOURCE_USERNAME`| `radar` | Yes | Database username. |
| `radar-control-plane` | `SPRING_DATASOURCE_PASSWORD`| `radar_password` | Yes | Database password. |
| `radar-control-plane` | `SERVER_PORT` | `8080` | No | Control plane listening port. |
| `postgres` | `POSTGRES_DB` | `prompt_radar` | Yes | Initial database name. |
| `postgres` | `POSTGRES_USER` | `radar` | Yes | Superuser username. |
| `postgres` | `POSTGRES_PASSWORD` | `radar_password` | Yes | Superuser password. |

---

## 4. Health Checks & Startup Ordering

Docker Compose enforces strict startup ordering using native Docker health checks:

1. **PostgreSQL** performs self-check:
   ```yaml
   healthcheck:
     test: ["CMD-SHELL", "pg_isready -U radar -d prompt_radar"]
     interval: 5s
     timeout: 5s
     retries: 5
   ```
2. **Spring Boot Control Plane** waits for PostgreSQL `service_healthy` before running Flyway database migrations and initializing connection pools.
3. **Go Proxy** starts and resolves the control plane host on the Docker bridge network.
4. **Angular UI** begins serving after the proxy is reachable.

---

## 5. Production Considerations

For production Kubernetes or cloud deployments:

### 1. Reverse Proxy & TLS Termination
Place an Ingress Controller (Nginx, Traefik, or AWS ALB) in front of the services:
* Route `/v1/*` to `radar-proxy:8081` with HTTP/2 enabled for multiplexed SSE.
* Route `/api/v1/*` to `radar-control-plane:8080`.
* Route `/*` to `radar-ui:80`.
* Enable gzip / brotli compression for static assets.

### 2. Horizontal Scaling & High Availability
* **Data Plane**: Because the Go proxy is stateless (with seed vectors loaded at boot), multiple `radar-proxy` replicas can sit behind a round-robin load balancer.
* **Control Plane**: Spring Boot instances can scale horizontally; batch telemetry insertions use standard PostgreSQL row inserts (`telemetry_logs`).
* **PostgreSQL**: Deploy managed PostgreSQL (AWS RDS / GCP Cloud SQL) with read replicas and automated backups.

### 3. Connection Pooling
Deploy **PgBouncer** in front of PostgreSQL to handle high-concurrency connection spikes from scaled application replicas.

### 4. Secrets Management
Never commit `GEMINI_API_KEY` or database passwords to version control. Pass secrets through Kubernetes Secrets, HashiCorp Vault, or AWS Secrets Manager.
