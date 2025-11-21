---
id: openssl-fail
type: game-over
---

# Certificate Problems

Using a quick openssl script seemed efficient, but...

```bash
openssl req -x509 -newkey rsa:2048 -nodes -out cert.pem
```

The certificates work initially, but you forgot to set the Subject Alternative Names (SANs) correctly. The kube-apiserver refuses connections:

```
x509: certificate is valid for localhost, not kubernetes.default
```

Your kubelet can't talk to the API server. Pod scheduling fails. Everything breaks.

This is why Kelsey recommends cfssl. You've learned the hard way why certificate configuration matters.

**Cluster Status: FAILED**
