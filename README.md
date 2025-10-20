# Endpoint Health Checker

A tool for quickly detecting the health status of backend pods for Kubernetes services.

When a node experiences a power outage, kubelet's `--node-monitor-grace-period` parameter defaults to 40s, which means service traffic will be routed to unavailable pods for up to 40 seconds. Since this parameter cannot be set too low (otherwise network fluctuations would cause nodes to be falsely reported as NotReady), this component is specifically designed to solve the rapid node failure detection problem.

## Features

- **Multiple Detection Methods**: Supports TCP port probing and ICMP probing
- **High Availability**: Based on Kubernetes Leader Election mechanism
- **Configurable Retry**: Supports custom retry count and timeout settings
- **Parallel Processing**: Uses worker thread pools for parallel health checks

## How It Works

1. Retrieves pods that require health checks
2. Performs parallel TCP port probing or ICMP probing
3. Retries specified number of times upon failure
  - With ports: TCP probing
  - Without ports: ICMP probing
  - Retry 10 times on failure, mark as Ready when successful
4. Updates Pod Ready status or readinessGates status

Leader Election ensures only one instance performs checks, avoiding duplicate work.

## Usage

### Configuration Methods

**Annotation Method** (Recommended): Pod Ready status is independent of health check results

```yaml
metadata:
  annotations:
    endpoint-health-checker.io/enabled: "true"
```

**ReadinessGates Method**: Pod becomes Ready only after successful health check

```yaml
spec:
  readinessGates:
    - conditionType: "endpointHealthCheckSuccess"
```

## Configuration Options

### Environment Variables

| Environment Variable | Default Value | Description |
|---------------------|---------------|-------------|
| `HEALTH_CHECK_INTERVAL` | `1s` | Health check interval |
| `HEALTH_CHECK_TIMEOUT` | `1s` | Single probe timeout |
| `HEALTH_CHECK_CONCURRENCY` | `10` | Number of concurrent worker threads |
| `HEALTH_CHECK_RETRY_COUNT` | `10` | Health check retry count |
| `LEASE_NAME` | `endpoint-health-checker-leader` | Leader election lease name |
| `LEASE_DURATION` | `4s` | Leader election lease duration |
| `RENEW_DEADLINE` | `2s` | Leader election renew deadline |
| `RETRY_PERIOD` | `500ms` | Leader election retry period |

## Deployment

### Online Helm Repository Deployment

```bash
# Add Helm Repository
helm repo add kubeovn-hc https://kubeovn.github.io/endpoint-health-checker/
helm repo update

# Install to kube-system namespace
helm install endpoint-health-checker kubeovn-hc/endpoint-health-checker --namespace kube-system

# Install with specific version
helm install endpoint-health-checker kubeovn-hc/endpoint-health-checker --version v0.1.0 --namespace kube-system
```

### Configuration Examples

**Annotation Method**: Pod Ready status is independent of health check results

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    metadata:
      annotations:
        endpoint-health-checker.io/enabled: "true"
    spec:
      containers:
        - name: my-app
          image: my-app:latest
          ports:
            - containerPort: 8080
```

**ReadinessGates Method**: Pod becomes Ready only after successful health check

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      readinessGates:
        - conditionType: "endpointHealthCheckSuccess"
      containers:
        - name: my-app
          image: my-app:latest
          ports:
            - containerPort: 8080
```
