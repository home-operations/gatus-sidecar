# 🚀 gatus-sidecar

A powerful Kubernetes sidecar that automatically generates [Gatus](https://github.com/TwiN/gatus) monitoring configuration from Kubernetes Ingress and Gateway API HTTPRoute resources. ⚡

## 🔍 Overview

gatus-sidecar is a lightweight Go application designed to run as a sidecar container alongside Gatus. It watches Kubernetes resources (Ingress or HTTPRoute) and automatically generates Gatus endpoint configurations, eliminating the need to manually maintain monitoring configurations for your web services. 🎯

## ✨ Features

- **🔄 Dual Mode Operation**: Supports both Kubernetes Ingress and Gateway API HTTPRoute resources
- **🔍 Automatic Discovery**: Watches for resource changes and dynamically updates monitoring configurations  
- **🎛️ Flexible Filtering**: Filter resources by namespace, gateway name, or ingress class
- **📋 Template Support**: Override default configurations using Kubernetes annotations
- **⚡ Zero Downtime**: Hot-reload configurations without restarting Gatus
- **🪶 Lightweight**: Minimal resource footprint with efficient Kubernetes API watching

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
| `--mode` | `httproute` | Mode to run in: `httproute` or `ingress` |
| `--namespace` | `""` | Namespace to watch (empty for all namespaces) |
| `--gateway` | `""` | Gateway name to filter HTTPRoutes |
| `--ingress-class` | `""` | Ingress class to filter Ingresses |
| `--output` | `/config` | Directory to write generated YAML files |
| `--default-interval` | `1m` | Default interval value for endpoints |
| `--default-dns` | `tcp://1.1.1.1:53` | Default DNS resolver for endpoints |
| `--default-condition` | `[STATUS] == 200` | Default condition for health checks |
| `--annotation-config` | `gatus.home-operations.com/endpoint` | Annotation key for YAML config override |

### 🌐 HTTPRoute Mode

Monitor Gateway API HTTPRoute resources:

```bash
gatus-sidecar --mode=httproute --gateway=my-gateway
```

### 🔀 Ingress Mode

Monitor Kubernetes Ingress resources:
```bash
gatus-sidecar --mode=ingress --ingress-class=nginx
```

## ⚙️ Configuration

### 🚀 Basic Endpoint Generation

The sidecar automatically generates Gatus endpoint configurations based on the hostnames found in your Kubernetes resources. For example, an HTTPRoute with hostname `api.example.com` would generate:

```yaml
endpoints:
  - name: "api.example.com"
    url: "https://api.example.com"
    interval: 1m
    dns:
      resolver: "tcp://1.1.1.1:53"
    conditions:
      - "[STATUS] == 200"
```

### 🏷️ Custom Configuration via Annotations

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
        - --mode=httproute
        - --gateway=my-gateway
        - --output=/config
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

### 🏗️ Building

```bash
go mod download
go build -o gatus-sidecar cmd/root.go
```

## 🏗️ Architecture

```text
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │  gatus-sidecar   │    │     Gatus       │
│   API Server    │◄───┤    Controller    ├───►│   Monitoring    │
│                 │    │                  │    │                 │
│ ▪ HTTPRoutes    │    │ ▪ Watches K8s    │    │ ▪ Reads config  │
│ ▪ Ingresses     │    │ ▪ Generates YAML │    │ ▪ Monitors URLs │
│                 │    │ ▪ Writes files   │    │ ▪ Sends alerts  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

The sidecar operates by:

1. **👀 Watching** Kubernetes resources via the API server
2. **⚡ Processing** resource events (create, update, delete)
3. **📝 Generating** Gatus configuration files in YAML format
4. **💾 Writing** files to a shared volume that Gatus reads from

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
