---
id: etcd-disaster
type: game-over
---

# etcd Configuration Failure

Running etcd entirely in memory? That's not even an option etcd supports in the way you're thinking.

You try various flags but etcd keeps persisting to disk. You get frustrated and set `--data-dir=/dev/null`:

```
panic: failed to create database: mkdir /dev/null: not a directory
```

Even if you could make this work, the first time you restart etcd, all cluster state vanishes. Every pod, every service, every secret - gone.

Kubernetes requires persistent etcd storage. This is non-negotiable.

**Cluster Status: FAILED**
