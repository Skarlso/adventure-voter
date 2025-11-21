---
id: network-broken
type: game-over
---

# Pod Network - Critical Failure

Without configuring pod networking, your nodes stay in NotReady state:

```
NAME       STATUS     ROLES    AGE   VERSION
worker-0   NotReady   <none>   5m    v1.28.0
worker-1   NotReady   <none>   5m    v1.28.0
```

The kubelet reports:

```
network plugin is not ready: cni config uninitialized
```

You try to create a deployment anyway:

```bash
kubectl create deployment nginx --image=nginx
```

The pod stays in ContainerCreating forever. It can't get an IP address. No CNI plugin means no pod networking.

Kubernetes refuses to run workloads without functioning networking. This is a fundamental requirement.

**Cluster Status: FAILED**
