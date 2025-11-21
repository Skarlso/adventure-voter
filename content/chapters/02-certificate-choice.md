---
id: certificate-choice
type: decision
timer: 30
question: What should we use for the Certificate Authority setup?
choices:
  - id: cfssl
    label: Use cfssl to generate certificates
    next: cfssl-success
  - id: openssl
    label: Use openssl with a quick script
    next: openssl-fail
  - id: self-signed
    label: Skip CA, use self-signed certs
    next: self-signed-disaster
---

# Certificate Authority Setup

First critical decision: we need to provision a Certificate Authority and generate TLS certificates for our cluster components.

What should we use?
