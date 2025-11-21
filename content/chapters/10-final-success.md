---
id: final-success
type: terminal
---

# Kubernetes The Hard Way - Complete!

Congratulations! You've successfully built a Kubernetes cluster from scratch.

Let's verify everything works:

```bash
kubectl run nginx --image=nginx --port=80
kubectl expose pod nginx --type=NodePort
kubectl get services
```

```
NAME    TYPE       CLUSTER-IP     EXTERNAL-IP   PORT(S)        AGE
nginx   NodePort   10.32.0.15     <none>        80:30080/TCP   5s
```

Your cluster is fully operational:
- Secure certificate infrastructure ✓
- Highly available etcd cluster ✓
- Properly configured API server ✓
- Working pod networking ✓

You've learned what Kelsey teaches: understanding the fundamentals makes you a better Kubernetes operator. When things break in production, you'll know why.

Thanks for participating in this interactive journey through Kubernetes The Hard Way!
