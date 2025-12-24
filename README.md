# 🚀 gatus-sidecar

A powerful Kubernetes sidecar that automatically generates [Gatus](https://github.com/TwiN/gatus) monitoring configuration from Kubernetes resources including Ingress, Gateway API HTTPRoute, and Service resources. ⚡

## 🔍 Overview

gatus-sidecar is a lightweight Go application designed to run as a sidecar container alongside Gatus. It watches multiple types of Kubernetes resources and automatically generates Gatus endpoint configurations, eliminating the need to manually maintain monitoring configurations for your web services and infrastructure. 🎯

## ✨ Features

- **🔄 Multi-Resource Support**: Supports Kubernetes Ingress, Gateway API HTTPRoute, and Service resources
- **🔍 Automatic Discovery**: Watches for resource changes and dynamically updates monitoring configurations  
- **🎛️ Flexible Filtering**: Filter resources by namespace, gateway name, or ingress class
- **📋 Template Support**: Override default configurations using Kubernetes annotations
- **🏗️ Gateway Inheritance**: HTTPRoutes automatically inherit annotations from their parent gateway
- **🏗️ Ingress Inheritance**: Ingress automatically inherit annotations from their parent ingress class
- **👥 Auto-Grouping**: Automatically group endpoints by namespace (Services) or gateway/ingress class (HTTPRoutes/Ingresses)
- **⚡ Zero Downtime**: Hot-reload configurations without restarting Gatus
- **🪶 Lightweight**: Minimal resource footprint with efficient Kubernetes API watching
- **🎯 Selective Processing**: Enable/disable monitoring per resource with annotations

## 📦 Installation

### 🐳 Using Docker

```bash
docker pull ghcr.io/home-operations/gatus-sidecar:latest
```

### 🔨 Building from Source

```bash
git clone https://github.com/home-operations/gatus-sidecar.git
cd gatus-sidecar
go build -o gatus-sidecar cmd/root.go
```

## 🛠️ Usage

### ⚙️ Command Line Options

```bash
gatus-sidecar [options]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--namespace` | `""` | Namespace to watch (empty for all namespaces) |
| `--gateway-name` | `""` | Gateway name to filter HTTPRoutes |
| `--ingress-class` | `""` | Ingress class to filter Ingresses |
| `--enable-httproute` | `false` | Enable HTTPRoute endpoint creation |
| `--enable-ingress` | `false` | Enable Ingress endpoint creation |
| `--enable-service` | `false` | Enable Service endpoint creation |
| `--auto-httproute` | `false` | Automatically create endpoints for HTTPRoutes |
| `--auto-ingress` | `false` | Automatically create endpoints for Ingresses |
| `--auto-service` | `false` | Automatically create endpoints for Services |
| `--output` | `/config/gatus-sidecar.yaml` | File to write generated YAML |
| `--default-interval` | `1m` | Default interval value for endpoints |
| `--annotation-config` | `gatus.home-operations.com/endpoint` | Annotation key for YAML config override |
| `--annotation-enabled` | `gatus.home-operations.com/enabled` | Annotation key for enabling/disabling resource processing |

### 🌐 HTTPRoute Mode

Monitor Gateway API HTTPRoute resources:

```bash
gatus-sidecar --auto-httproute --gateway-name=my-gateway
```

### 🔀 Ingress Mode

Monitor Kubernetes Ingress resources:

```bash
gatus-sidecar --auto-ingress --ingress-class=nginx
```

### 🔧 Service Mode

Monitor Kubernetes Service resources:

```bash
gatus-sidecar --auto-service --namespace=production
```

### 📊 Multi-Resource Mode

Monitor all resource types simultaneously:

```bash
gatus-sidecar --auto-httproute --auto-ingress --auto-service
```

## ⚙️ Configuration

### 🚀 Basic Endpoint Generation

The sidecar automatically generates Gatus endpoint configurations based on the resources found in your Kubernetes cluster:

**HTTPRoute Example**: An HTTPRoute with hostname `api.example.com` would generate:

```yaml
endpoints:
  - name: "api-route"
    url: "https://api.example.com"
    interval: 1m
```

**Service Example**: A Service named `database` would generate:

```yaml
endpoints:
  - name: "database"
    url: "tcp://database.default.svc:5432"
    interval: 1m
    conditions:
      - "[CONNECTED] == true"
```

### 🏷️ Resource Selection Modes

The sidecar can operate in different modes based on your needs:

1. **Automatic Mode**: Use `--auto-httproute`, `--auto-ingress`, or `--auto-service` to automatically process all resources of that type
2. **Annotation-Based**: Without the auto flags, only resources with specific annotations are processed
3. **Hybrid Mode**: Combine both approaches for fine-grained control

### 🎯 Selective Processing with Annotations

Control which resources should be monitored using the enabled annotation:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api-route
  annotations:
    gatus.home-operations.com/enabled: "true"  # Enable monitoring
spec:
  # ... HTTPRoute spec
```

Set to `"false"` or `"0"` to disable monitoring for a specific resource.

### 🏗️ Gateway Inheritance

HTTPRoutes automatically inherit annotations from their parent Gateway. This allows you to set common monitoring configurations at the Gateway level:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: production-gateway
  annotations:
    gatus.home-operations.com/endpoint: |
      interval: 30s
      alerts:
        - type: slack
          webhook-url: "https://hooks.slack.com/..."
spec:
  # ... Gateway spec
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api-route
  annotations:
    gatus.home-operations.com/endpoint: |
      conditions:
        - "[STATUS] == 200"
        - "[RESPONSE_TIME] < 500"
spec:
  parentRefs:
    - name: production-gateway
  hostnames:
    - api.example.com
  # ... rest of HTTPRoute spec
```

### 🏗️ Ingress Inheritance

Ingresses automatically inherit annotations from their parent IngressClass. This allows you to set common monitoring configurations at the IngressClass level:

```yaml
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: internal
  annotations:
    gatus.home-operations.com/endpoint: |
      interval: 30s
      alerts:
        - type: slack
          webhook-url: "https://hooks.slack.com/..."
spec:
  # ... IngressClass spec
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingressapp
  annotations:
    gatus.home-operations.com/endpoint: |
      conditions:
        - "[STATUS] == 200"
        - "[RESPONSE_TIME] < 500"
spec:
  ingressClassName: internal
  rules:
  # ... rest of Ingress spec
```

The resulting endpoint will have both IngressClass and Ingress configurations merged.

### � Custom Configuration via Annotations

You can override the default configuration by adding the annotation specified in `--annotation-config` to your Kubernetes resources:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api-route
  annotations:
    gatus.home-operations.com/endpoint: |
      interval: 30s
      conditions:
        - "[STATUS] == 200"
        - "[RESPONSE_TIME] < 500"
      alerts:
        - type: slack
          webhook-url: "https://hooks.slack.com/..."
spec:
  hostnames:
    - api.example.com
  # ... rest of HTTPRoute spec
```

#### 🔧 Service Monitoring Example

```yaml
apiVersion: v1
kind: Service
metadata:
  name: redis
  annotations:
    gatus.home-operations.com/enabled: "true"
    gatus.home-operations.com/endpoint: |
      interval: 15s
      conditions:
        - "[CONNECTED] == true"
        - "[RESPONSE_TIME] < 100"
spec:
  ports:
    - port: 6379
  # ... rest of Service spec
```

#### 🌐 Ingress Monitoring Example

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web-app
  annotations:
    kubernetes.io/ingress.class: "nginx"
    gatus.home-operations.com/endpoint: |
      interval: 2m
      conditions:
        - "[STATUS] == 200"
        - "[CERTIFICATE_EXPIRATION] > 72h"
spec:
  rules:
    - host: webapp.example.com
  # ... rest of Ingress spec
```

## 🚀 Deployment Examples

### ☸️ Kubernetes Deployment with Gatus

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gatus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gatus
  template:
    metadata:
      labels:
        app: gatus
    spec:
      serviceAccountName: gatus-sidecar  # Required for Kubernetes API access
      containers:
      - name: gatus
        image: ghcr.io/twin/gatus:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: gatus-config
          mountPath: /config
      - name: gatus-sidecar
        image: ghcr.io/home-operations/gatus-sidecar:latest
        args:
        - --auto-httproute
        - --auto-ingress
        - --auto-service
        - --auto-group
        - --gateway-name=production-gateway
        - --output=/config/gatus-sidecar.yaml
        volumeMounts:
        - name: gatus-config
          mountPath: /config
      volumes:
      - name: gatus-config
        emptyDir: {}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: gatus-sidecar
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gatus-sidecar
rules:
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses", "ingressclasses"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["gateway.networking.k8s.io"]
  resources: ["httproutes", "gateways"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: gatus-sidecar
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gatus-sidecar
subjects:
- kind: ServiceAccount
  name: gatus-sidecar
  namespace: default  # Update to your deployment namespace
```

### 🎯 Namespace-Scoped Deployment

For monitoring resources in a specific namespace:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gatus
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gatus
  template:
    metadata:
      labels:
        app: gatus
    spec:
      containers:
      - name: gatus
        image: ghcr.io/twin/gatus:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: gatus-config
          mountPath: /config
      - name: gatus-sidecar
        image: ghcr.io/home-operations/gatus-sidecar:latest
        args:
        - --namespace=production
        - --auto-httproute
        - --auto-service
        - --output=/config/gatus-sidecar.yaml
        volumeMounts:
        - name: gatus-config
          mountPath: /config
      volumes:
      - name: gatus-config
        emptyDir: {}
```

## 💻 Development

### 📋 Prerequisites

- Go 1.25 or later
- Kubernetes cluster access
- kubectl configured
- Access to Gateway API CRDs (if using HTTPRoute monitoring)

### 🏗️ Building

```bash
go mod download
go build -o gatus-sidecar cmd/root.go
```

### 🧪 Testing

The project includes comprehensive controller logic for handling multiple resource types:

- **HTTPRoute Controller**: Monitors Gateway API HTTPRoute resources with parent Gateway annotation inheritance
- **Ingress Controller**: Monitors traditional Kubernetes Ingress resources with parent IngressClass annotation inheritance
- **Service Controller**: Monitors Kubernetes Service resources for infrastructure monitoring
- **State Manager**: Centralizes endpoint state management and YAML generation

### 🛠️ Local Development

To run the sidecar locally against a Kubernetes cluster:

```bash
# Build the binary
go build -o gatus-sidecar cmd/root.go

# Run with auto-discovery enabled (requires KUBECONFIG)
./gatus-sidecar --auto-httproute --auto-service --output=./gatus-config.yaml

# Or run with selective monitoring
./gatus-sidecar --namespace=default --gateway-name=my-gateway
```

## 🏗️ Architecture

```text
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │  gatus-sidecar   │    │     Gatus       │
│   API Server    │◄───┤   Controllers    ├───►│   Monitoring    │
│                 │    │                  │    │                 │
│ ▪ HTTPRoutes    │    │ ▪ Watches K8s    │    │ ▪ Reads config  │
│ ▪ Gateways      │    │ ▪ Merges configs │    │ ▪ Monitors URLs │
│ ▪ Ingresses     │    │ ▪ Generates YAML │    │ ▪ Sends alerts  │
│ ▪ Services      │    │ ▪ Writes files   │    │ ▪ Health checks │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

The sidecar operates by:

1. **👀 Watching** Multiple Kubernetes resource types via the API server
2. **🔗 Inheriting** Annotations from parent resources (Gateway → HTTPRoute, IngressClass → Ingress)
3. **⚡ Processing** Resource events (create, update, delete) in real-time
4. **🎛️ Filtering** Resources based on configuration flags and annotations
5. **📝 Generating** Gatus configuration files in YAML format
6. **💾 Writing** Files to a shared volume that Gatus reads from
7. **🔄 Updating** Configurations dynamically without Gatus restarts

### 🎯 Resource Processing Logic

```text
┌─────────────┐    ┌────────────────┐    ┌──────────────┐
│  Resource   │───►│ Annotation     │───►│   Endpoint   │
│  Discovery  │    │ Processing     │    │  Generation  │
│             │    │                │    │              │
│ ▪ Auto Mode │    │ ▪ Parent merge │    │ ▪ URL extract│
│ ▪ Selective │    │ ▪ Template     │    │ ▪ Conditions │
│ ▪ Filtered  │    │ ▪ Enabled      │    │ ▪ Grouping   │
└─────────────┘    └────────────────┘    └──────────────┘
```

## 🤝 Contributing

1. Fork the repository 🍴
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request 🎉

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Related Projects

- [Gatus](https://github.com/TwiN/gatus) - The monitoring solution this sidecar supports 📊
- [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) - Next generation of Kubernetes ingress ⚡
