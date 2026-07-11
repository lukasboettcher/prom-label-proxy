# Docker Compose multi-label example

This example runs Prometheus with Docker Compose. Prometheus scrapes itself twice with
these label sets:

| tenant | cluster | environment |
| --- | --- | --- |
| `team-a` | `eu-1` | `production` |
| `team-b` | `us-1` | `staging` |

From the repository root, start Prometheus:

```bash
docker compose -f examples/prometheus/compose.yaml up
```

In another terminal, run prom-label-proxy:

```bash
go run . \
  -insecure-listen-address=:8080 \
  -upstream=http://localhost:9090 \
  -label=tenant -header-name=X-Tenant \
  -label=cluster -header-name=X-Cluster \
  -label=environment -header-name=X-Environment \
  -error-on-replace
```

Wait a few seconds for the first scrapes, then query the first label set through
prom-label-proxy:

```bash
curl -G 'http://localhost:8080/api/v1/query' \
  --data-urlencode 'query=up{job="local-demo"}' \
  -H 'X-Tenant: team-a' \
  -H 'X-Cluster: eu-1' \
  -H 'X-Environment: production'
```

The response contains one series. The second label set works in the same way:

```bash
curl -G 'http://localhost:8080/api/v1/query' \
  --data-urlencode 'query=up{job="local-demo"}' \
  -H 'X-Tenant: team-b' \
  -H 'X-Cluster: us-1' \
  -H 'X-Environment: staging'
```

A mixed label set is valid but cannot match either series, so the result is
empty:

```bash
curl -G 'http://localhost:8080/api/v1/query' \
  --data-urlencode 'query=up{job="local-demo"}' \
  -H 'X-Tenant: team-a' \
  -H 'X-Cluster: us-1' \
  -H 'X-Environment: production'
```

Omitting any required header returns HTTP 400:

```bash
curl -i -G 'http://localhost:8080/api/v1/query' \
  --data-urlencode 'query=up{job="local-demo"}' \
  -H 'X-Tenant: team-a' \
  -H 'X-Cluster: eu-1'
```

The example enables `-error-on-replace`, so a conflicting matcher also returns
HTTP 400:

```bash
curl -i -G 'http://localhost:8080/api/v1/query' \
  --data-urlencode 'query=up{tenant="team-b"}' \
  -H 'X-Tenant: team-a' \
  -H 'X-Cluster: eu-1' \
  -H 'X-Environment: production'
```

Prometheus is available directly at <http://localhost:9090> for comparison.
Direct access bypasses label enforcement.
