---
id: network-success
type: story
next: final-success
---

# Pod Network - Success!

You install Weave Net as the CNI plugin:

```bash
kubectl apply -f \
  "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')"
```

Within seconds:

```
NAME           STATUS   ROLES    AGE   VERSION
worker-0       Ready    <none>   2m    v1.28.0
worker-1       Ready    <none>   2m    v1.28.0
```

Your nodes transition to Ready state. Pods can communicate across nodes. DNS works. Services route correctly.

Your cluster is fully functional. Time to deploy some applications!

**Cluster Status: SUCCESS! 🎉**
