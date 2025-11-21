---
id: apiserver-choice
type: decision
timer: 30
question: How should we configure the kube-apiserver?
choices:
  - id: proper-flags
    label: Use all recommended security flags
    next: apiserver-success
  - id: skip-auth
    label: Disable authentication for easier testing
    next: apiserver-insecure
  - id: wrong-port
    label: Change default port to avoid conflicts
    next: apiserver-broken
---

# kube-apiserver Configuration

Time to start the Kubernetes API server. This is the central management component that everything talks to.

How should we configure it?
