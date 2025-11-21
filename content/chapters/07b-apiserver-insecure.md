---
id: apiserver-insecure
type: story
next: network-choice
---

# API Server - Dangerously Insecure

You disabled authentication with `--insecure-port=8080`:

```bash
kube-apiserver --insecure-port=8080 --authorization-mode=AlwaysAllow
```

The API server starts. Everything works. Anyone can do anything without credentials.

```bash
curl http://localhost:8080/api/v1/namespaces/kube-system/secrets
```

All secrets are readable. Anyone can create pods, delete namespaces, access everything.

In 2019, Tesla's Kubernetes cluster was cryptomined because of exactly this misconfiguration. The insecure port is now deprecated for good reason.

**Cluster Status: WORKING (but completely compromised)**
