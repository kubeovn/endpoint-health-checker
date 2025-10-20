# Endpoint Health Checker

快速检测 Kubernetes service 后端 Pod 健康状态的工具。

当节点断电时，kubelet 的 `--node-monitor-grace-period` 参数默认为 40s，这意味着 Service 流量会被路由到不可用的 Pod 上长达 40 秒。由于这个参数不能设置过低（否则网络抖动会导致节点误报 NotReady），因此该组件专门用于解决这个节点故障快速检测问题。

## 特性

- **多种探测方式**：支持 TCP 端口探测和 ICMP 探测
- **高可用性**：基于 Kubernetes Leader Election 机制
- **可配置重试**：支持自定义重试次数和超时时间
- **并行处理**：使用工作线程池并行执行健康检查

## 工作原理

1. 获取需要健康检查的 Pod
2. 并行执行 TCP 端口探测或 ICMP 探测
3. 失败后重试指定次数
  - 有端口：TCP 探测
  - 无端口：ICMP 探测
  - 失败重试 10 次，成功后标记为 Ready
4. 更新 Pod 的 Ready 或者 readinessGates 状态

通过 Leader Election 确保只有一个实例执行检查，避免重复工作。

## 使用方式

### 配置方式

**Annotation 方式**（推荐）：Pod Ready 状态独立于健康检查结果

```yaml
metadata:
  annotations:
    endpoint-health-checker.io/enabled: "true"
```

**ReadinessGates 方式**：Pod 需健康检查成功才变为 Ready

```yaml
spec:
  readinessGates:
    - conditionType: "endpointHealthCheckSuccess"
```

## 配置选项

### 环境变量

| 环境变量 | 默认值 | 描述 |
|----------|--------|------|
| `HEALTH_CHECK_INTERVAL` | `1s` | 健康检查间隔 |
| `HEALTH_CHECK_TIMEOUT` | `1s` | 单次探测超时时间 |
| `HEALTH_CHECK_CONCURRENCY` | `10` | 并发工作线程数量 |
| `HEALTH_CHECK_RETRY_COUNT` | `10` | 健康检查重试次数 |
| `LEASE_NAME` | `endpoint-health-checker-leader` | Leader election 租约名称 |
| `LEASE_DURATION` | `4s` | Leader election 租约持续时间 |
| `RENEW_DEADLINE` | `2s` | Leader election 续约截止时间 |
| `RETRY_PERIOD` | `500ms` | Leader election 重试周期 |

## 部署

### 在线 Helm Repository 部署

```bash
# 添加 Helm Repository
helm repo add kubeovn-hc https://kubeovn.github.io/endpoint-health-checker/
helm repo update

# 安装到 kube-system 命名空间
helm install endpoint-health-checker kubeovn-hc/endpoint-health-checker --namespace kube-system

# 或者安装到自定义命名空间
helm install endpoint-health-checker kubeovn-hc/endpoint-health-checker --namespace endpoint-health-checker --create-namespace
```

### 配置示例

**Annotation 方式**：Pod Ready 状态独立于健康检查结果

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

**ReadinessGates 方式**：Pod 需健康检查成功才变为 Ready

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
