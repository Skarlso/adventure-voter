---
id: etcd-warning
type: story
next: apiserver-choice
---

# etcd Single Node - Risky

A single etcd instance works for this demo, but it's not production-ready.

```bash
etcd --name controller-0 --data-dir=/var/lib/etcd
```

The cluster starts successfully. Everything works. But there's no fault tolerance.

If this etcd instance crashes, you lose all cluster state:
- All pod definitions
- All secrets
- All configuration
- Everything

For a presentation this is fine. For production, you'd be fired.

**Cluster Status: WORKING (but fragile)**
