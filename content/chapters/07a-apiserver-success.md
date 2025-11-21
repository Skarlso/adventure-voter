---
id: apiserver-success
type: story
next: network-choice
---

# API Server - Secure and Running

Perfect! Using proper security flags:

```bash
kube-apiserver \
  --advertise-address=10.240.0.10 \
  --allow-privileged=true \
  --authorization-mode=Node,RBAC \
  --client-ca-file=/var/lib/kubernetes/ca.pem \
  --enable-admission-plugins=NodeRestriction \
  --etcd-servers=https://127.0.0.1:2379 \
  --service-cluster-ip-range=10.32.0.0/24 \
  --tls-cert-file=/var/lib/kubernetes/kubernetes.pem \
  --tls-private-key-file=/var/lib/kubernetes/kubernetes-key.pem
```

The API server starts successfully. RBAC is enabled. TLS is configured. Admission controllers are active.

This is how you run Kubernetes in production.
