---
id: etcd-success
type: story
next: apiserver-choice
---

# etcd Cluster - Production Ready

Excellent! A three-node etcd cluster with TLS is the correct approach.

```bash
etcd --name controller-0 \
  --cert-file=/etc/etcd/kubernetes.pem \
  --key-file=/etc/etcd/kubernetes-key.pem \
  --peer-cert-file=/etc/etcd/kubernetes.pem \
  --peer-key-file=/etc/etcd/kubernetes-key.pem \
  --trusted-ca-file=/etc/etcd/ca.pem \
  --peer-trusted-ca-file=/etc/etcd/ca.pem
```

Your etcd cluster achieves quorum. Data replicates across three nodes. If one node fails, the cluster survives.

This is production-grade configuration. Your cluster state is safe.
