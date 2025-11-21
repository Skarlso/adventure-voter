---
id: network-choice
type: decision
timer: 30
question: What's your approach to pod networking?
choices:
  - id: cni-plugin
    label: Install a proper CNI plugin (Calico/Weave)
    next: network-success
  - id: host-network
    label: Use host networking for all pods
    next: network-mess
  - id: skip-network
    label: Configure networking later
    next: network-broken
---

# Pod Network Configuration

The control plane is running. Now we need pod networking. Without this, pods can't communicate.

What's your approach?
