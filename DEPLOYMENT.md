# 🚢 Hetzner Deployment Guide

Quick guide to deploy Kubernetes Quest to a Hetzner VPS.

## 1️⃣ Create Hetzner Server

1. Go to [Hetzner Cloud Console](https://console.hetzner.cloud/)
2. Create new project
3. Add server:
   - **Image**: Ubuntu 22.04
   - **Type**: CPX11 (2 vCPU, 2 GB RAM) - €4.15/month
   - **Location**: Your choice
   - **SSH Key**: Add your public key

## 2️⃣ Initial Server Setup

```bash
# SSH into your server
ssh root@YOUR_SERVER_IP

# Update system
apt update && apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Install Docker Compose
apt install docker-compose -y

# Install Git
apt install git -y

# Create app user
useradd -m -s /bin/bash kubequest
usermod -aG docker kubequest
```

## 3️⃣ Deploy Application

```bash
# Switch to app user
su - kubequest

# Clone repository
git clone https://github.com/YOUR_USERNAME/kube_adventures.git
cd kube_adventures/voting

# Build and run with Docker Compose
docker-compose up -d

# Check if running
docker-compose ps
curl http://localhost:8080/voter/
```

## 4️⃣ Setup Nginx Reverse Proxy

```bash
# Exit app user, back to root
exit

# Install Nginx
apt install nginx -y

# Create Nginx config
cat > /etc/nginx/sites-available/kubequest << 'EOF'
server {
    listen 80;
    server_name YOUR_DOMAIN_OR_IP;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

# Enable site
ln -s /etc/nginx/sites-available/kubequest /etc/nginx/sites-enabled/
rm /etc/nginx/sites-enabled/default  # Remove default site

# Test and reload Nginx
nginx -t
systemctl reload nginx
```

## 5️⃣ Setup SSL (Optional but Recommended)

```bash
# Install Certbot
apt install certbot python3-certbot-nginx -y

# Get SSL certificate (replace with your domain)
certbot --nginx -d YOUR_DOMAIN

# Certbot will auto-configure Nginx for HTTPS
```

## 6️⃣ Setup Firewall

```bash
# Allow SSH, HTTP, HTTPS
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

## 7️⃣ Auto-start on Reboot

```bash
# Docker Compose should auto-restart containers
# Verify restart policy in docker-compose.yml:
# restart: unless-stopped

# Enable Docker to start on boot
systemctl enable docker
```

## 📝 Update Your Application

```bash
# SSH to server
ssh kubequest@YOUR_SERVER_IP

# Pull latest changes
cd ~/kube_adventures/voting
git pull

# Rebuild and restart
docker-compose down
docker-compose up -d --build

# Check logs
docker-compose logs -f
```

## 🔍 Monitoring and Logs

```bash
# View logs
docker-compose logs -f

# Check container status
docker-compose ps

# Restart containers
docker-compose restart

# View Nginx logs
tail -f /var/log/nginx/access.log
tail -f /var/log/nginx/error.log
```

## 🎯 Access Your Presentation

```
http://YOUR_DOMAIN/
http://YOUR_DOMAIN/presenter/
http://YOUR_DOMAIN/voter/
```

## 📱 Generate QR Code

For easy audience access, generate a QR code:

```bash
# Using qrencode
apt install qrencode
echo "http://YOUR_DOMAIN/voter/" | qrencode -t PNG -o voter-qr.png

# Download and display during presentation
```

Or use online tools:
- https://www.qr-code-generator.com/
- https://goqr.me/

## 💰 Cost Estimate

**Monthly costs:**
- Server (CPX11): €4.15
- Domain (optional): ~€10-15/year
- SSL Certificate: Free (Let's Encrypt)

**Total: ~€5-6/month**

## 🐛 Troubleshooting

### Can't connect to server
```bash
# Check if Docker containers are running
docker-compose ps

# Check Nginx status
systemctl status nginx

# Check firewall
ufw status
```

### WebSocket connection failed
```bash
# Verify Nginx WebSocket config
cat /etc/nginx/sites-enabled/kubequest | grep -A 2 "Upgrade"

# Should show:
# proxy_set_header Upgrade $http_upgrade;
# proxy_set_header Connection "upgrade";
```

### Application not starting
```bash
# Check Docker logs
docker-compose logs

# Check disk space
df -h

# Check memory
free -h
```

## 🔐 Security Best Practices

1. **Don't use root** - Always use app user
2. **Enable firewall** - Only allow needed ports
3. **Use SSL** - Always use HTTPS in production
4. **Keep updated** - Regularly update system and Docker
5. **Strong passwords** - Use SSH keys, disable password login
6. **Backup** - Regular backups of content directory

```bash
# Disable password SSH login (SSH keys only)
nano /etc/ssh/sshd_config
# Set: PasswordAuthentication no
systemctl restart sshd
```

## 📦 Backup Strategy

```bash
# Backup content directory
tar -czf backup-$(date +%Y%m%d).tar.gz /home/kubequest/kube_adventures/voting/content/

# Download backup
scp kubequest@YOUR_SERVER_IP:~/backup-*.tar.gz ./

# Restore backup
tar -xzf backup-YYYYMMDD.tar.gz
```

## 🎬 Presentation Day Checklist

- [ ] Server is running and accessible
- [ ] SSL certificate is valid
- [ ] Test voter URL on phone
- [ ] Content is up to date
- [ ] QR code generated
- [ ] Backup server info saved
- [ ] Monitoring enabled

## 📞 Emergency Contacts

Keep handy:
- Server IP: _______________
- SSH user: kubequest
- Admin email: _______________
- Hetzner login: _______________

---

**Pro tip**: Deploy a day before your presentation to catch any issues early! 🚀
