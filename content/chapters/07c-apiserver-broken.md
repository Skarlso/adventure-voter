---
id: apiserver-broken
type: game-over
---

# API Server - Port Misconfiguration

You changed the secure port from 6443 to 8443 to avoid conflicts:

```bash
kube-apiserver --secure-port=8443 ...
```

The API server starts on port 8443. But now nothing can find it:

```
kubectl get nodes
The connection to the server localhost:6443 was refused
```

The kubelet is configured to connect to port 6443. The kube-scheduler expects port 6443. The kube-controller-manager looks for port 6443.

You'd need to reconfigure every component. The default port exists for a reason.

**Cluster Status: FAILED (API server unreachable)**
