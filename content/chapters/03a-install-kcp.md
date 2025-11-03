---
id: install-kcp
type: story
---

# Chapter 2: The KCP Adventure Begins 🚀

**Bold choice!** You decide to try KCP (Kubernetes Control Plane).

You navigate to [kcp.io](https://www.kcp.io/) and start reading the docs. It's described as:

> *"A Kubernetes-like control plane for workloads on many clusters"*

Sounds perfect for development! Let's get it installed.

```bash
# Download the latest KCP release
wget https://github.com/kcp-dev/kcp/releases/download/v0.11.0/kcp_0.11.0_linux_amd64.tar.gz

# Extract it
tar -xzf kcp_0.11.0_linux_amd64.tar.gz

# Run KCP
./bin/kcp start
```

The terminal springs to life with logs:

```
I1102 10:15:33.123456    1 server.go:123] Starting kcp server...
I1102 10:15:34.789012    1 server.go:456] kcp server started successfully
I1102 10:15:34.789123    1 server.go:789] Listening on https://localhost:6443
```

🎉 **Success!** KCP is running!

---

### Stats Update:
- ⏰ **Time Remaining**: 1.5 hours (-30 min)
- 🧠 **Sanity**: 95% (pretty good!)
- ☕ **Coffee Level**: Medium
- 💼 **Manager Patience**: Still good

---

*KCP is now running, but you need to configure kubectl to talk to it...*
