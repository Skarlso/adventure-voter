---
id: cfssl-success
type: story
next: etcd-choice
---

# Certificate Authority - Success!

Great choice! Using cfssl gives us a proper CA infrastructure.

```bash
cfssl gencert -initca ca-csr.json | cfssljson -bare ca
```

We now have:
- ca-key.pem (CA private key)
- ca.pem (CA certificate)

All our cluster components can now trust certificates signed by this CA. The kube-apiserver, kubelet, and etcd will communicate securely.

This is exactly how production clusters should be configured.
