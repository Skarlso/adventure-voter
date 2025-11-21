---
id: etcd-choice
type: decision
timer: 30
question: How should we configure etcd?
choices:
  - id: three-node
    label: Three-node etcd cluster with proper TLS
    next: etcd-success
  - id: single-node
    label: Single etcd instance (faster setup)
    next: etcd-warning
  - id: memory
    label: Run etcd in memory for performance
    next: etcd-disaster
---

# etcd Configuration

With certificates ready, we need to set up etcd, Kubernetes' distributed key-value store. How should we configure it?
