---
id: network-mess
type: game-over
---

# Pod Network - Host Networking Chaos

Using host networking for all pods means they share the node's network namespace:

```yaml
spec:
  hostNetwork: true
```

This creates immediate problems:

- Pods can't have conflicting ports (only one nginx on port 80 per node)
- No network isolation between pods
- No network policies work
- Pod IPs are actually node IPs
- Services don't work correctly

Your first deployment fails because you try to run 3 replicas of nginx, but they all try to bind to port 80 on the same node.

This defeats the entire purpose of Kubernetes networking.

**Cluster Status: SEVERELY LIMITED**
