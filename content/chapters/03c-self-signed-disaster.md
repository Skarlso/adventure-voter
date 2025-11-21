---
id: self-signed-disaster
type: game-over
---

# Security Disaster

Skipping the CA entirely? That's... bold.

Without a proper Certificate Authority, each component generates its own self-signed certificate. They can't verify each other's identity.

```
etcd: tls: failed to verify certificate
kube-apiserver: x509: certificate signed by unknown authority
kubelet: remote error: tls: bad certificate
```

Every component refuses to communicate. Your cluster is completely broken before it even starts.

In production, this would be a security audit nightmare. Mutual TLS authentication is fundamental to Kubernetes security.

**Cluster Status: CATASTROPHICALLY FAILED**
