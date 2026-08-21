# Deployment Guide

## Option 1: Docker / Podman (Recommended)

Docker is the easiest and most portable way to deploy Last War Alliance Manager. No Go, GCC, or Tesseract installation needed on the host.

### Prerequisites

- Docker Engine ≥ 24 **or** Podman ≥ 4 with `podman-compose`
- A domain name pointing to your server (for HTTPS)

### 1. Get the Compose File

The production compose file pulls the published image from GitHub Container Registry — no Go, GCC, or Tesseract needed. Either clone the repository:

```bash
git clone <your-repo-url> /opt/lastwar
cd /opt/lastwar
```

or download just `docker-compose.yml` from the repository into an empty directory.

### 2. Generate a Session Key

```bash
openssl rand -hex 32
```

### 3. Configure the Session Key

Create a `.env` file next to `docker-compose.yml` with the value from the previous step (Compose reads it automatically):

```bash
echo "SESSION_KEY=your-64-char-hex-key-here" > .env
chmod 600 .env
```

Alternatively edit `docker-compose.yml` and set `SESSION_KEY` directly:

```yaml
environment:
  DATABASE_PATH: /data/alliance.db
  SESSION_KEY: "your-64-char-hex-key-here"
```

### 4. Start the Application

```bash
# Docker
docker compose up -d

# Podman
podman-compose up -d
```

The database is persisted in `./data/alliance.db` on the host.

### 5. Set Up a Reverse Proxy with HTTPS

#### Option A: Caddy (Automatic HTTPS — Recommended)

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install caddy
```

Create `/etc/caddy/Caddyfile`:

```
your-domain.com {
    reverse_proxy localhost:8080

    header {
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
    }

    log {
        output file /var/log/caddy/lastwar-access.log {
            roll_size 100mb
            roll_keep 5
        }
    }
}
```

```bash
sudo systemctl enable --now caddy
sudo systemctl reload caddy
```

#### Option B: Nginx + Certbot

See the Nginx section later in this document for the full configuration.

### 6. Useful Docker Commands

```bash
# View logs
docker compose logs -f

# Stop the service
docker compose down

# Restart
docker compose restart

# Rebuild after a code change
docker compose up -d --build

# Open a shell inside the running container
docker compose exec lastwar sh

# Manual database backup
docker compose exec lastwar sh -c 'cp /data/alliance.db /data/alliance_backup_$(date +%Y%m%d_%H%M%S).db'
```

### 7. Automated Backups (Docker)

Add a cron job on the host to copy the database file out of the volume:

```bash
sudo tee /usr/local/bin/backup-lastwar.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/var/backups/lastwar"
mkdir -p "$BACKUP_DIR"
cp /opt/lastwar/data/alliance.db "$BACKUP_DIR/alliance_$(date +%Y%m%d_%H%M%S).db"
find "$BACKUP_DIR" -name "alliance_*.db" -mtime +7 -delete
EOF
sudo chmod +x /usr/local/bin/backup-lastwar.sh

# Run daily at 02:00
echo "0 2 * * * root /usr/local/bin/backup-lastwar.sh" | sudo tee /etc/cron.d/lastwar-backup
```

### 8. Updating the Application (Docker)

```bash
cd /opt/lastwar
git pull
docker compose up -d --build
```

---

## Option 2: Bare-Metal — Debian Server Deployment Guide

## Quick Setup with Caddy (Recommended - Automatic HTTPS)

Caddy automatically handles Let's Encrypt certificates with zero configuration.

### 1. Install Go on Debian

```bash
# Install Go 1.23 or higher
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

### 2. Install Build Dependencies

```bash
sudo apt update
sudo apt install -y gcc g++ build-essential tesseract-ocr tesseract-ocr-eng libtesseract-dev libleptonica-dev
```

### 3. Deploy Application

```bash
# Create application directory
sudo mkdir -p /opt/lastwar
sudo chown $USER:$USER /opt/lastwar

# Upload your files to /opt/lastwar
# (use scp, rsync, or git clone)

cd /opt/lastwar

# Build the application
go build -o alliance-manager main.go

# Create data directory
sudo mkdir -p /var/lib/lastwar
sudo chown $USER:$USER /var/lib/lastwar
```

### 4. Create Systemd Service

Create `/etc/systemd/system/lastwar.service`:

```ini
[Unit]
Description=Last War Alliance Manager
After=network.target

[Service]
Type=simple
User=lastwar
Group=lastwar
WorkingDirectory=/opt/lastwar
Environment="DATABASE_PATH=/var/lib/lastwar/alliance.db"
Environment="SESSION_KEY=replace-with-output-of-openssl-rand-hex-32"
Environment="PRODUCTION=true"
Environment="HTTPS=true"
ExecStart=/opt/lastwar/alliance-manager
Restart=always
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/lastwar
CapabilityBoundingSet=
AmbientCapabilities=
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM

[Install]
WantedBy=multi-user.target
```

### 5. Create Dedicated User

```bash
sudo useradd -r -s /bin/false -d /opt/lastwar lastwar
sudo chown -R lastwar:lastwar /opt/lastwar
sudo chown -R lastwar:lastwar /var/lib/lastwar
```

### 6. Install Caddy

```bash
# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy
```

### 7. Configure Caddy

Create `/etc/caddy/Caddyfile`:

```
your-domain.com {
    reverse_proxy localhost:8080
    
    # Security headers
    header {
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
    }
    
    # Rate limiting (requires caddy-rate-limit plugin)
    # rate_limit {
    #     zone dynamic {
    #         key {remote_host}
    #         events 100
    #         window 1m
    #     }
    # }
    
    # Logging
    log {
        output file /var/log/caddy/lastwar-access.log {
            roll_size 100mb
            roll_keep 5
        }
    }
}
```

Replace `your-domain.com` with your actual domain.

### 8. Configure Firewall

```bash
# Install and configure UFW
sudo apt install -y ufw

# Allow SSH (IMPORTANT - do this first!)
sudo ufw allow 22/tcp

# Allow HTTP and HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Enable firewall
sudo ufw enable
sudo ufw status
```

### 9. Start Services

```bash
# Start application
sudo systemctl daemon-reload
sudo systemctl enable lastwar
sudo systemctl start lastwar
sudo systemctl status lastwar

# Start Caddy (automatically gets Let's Encrypt cert)
sudo systemctl enable caddy
sudo systemctl restart caddy
sudo systemctl status caddy
```

That's it! Caddy will automatically obtain and renew Let's Encrypt certificates.

---

## Alternative: Nginx with Certbot

If you prefer Nginx:

### 1. Install Nginx and Certbot

```bash
sudo apt install -y nginx certbot python3-certbot-nginx
```

### 2. Configure Nginx

Create `/etc/nginx/sites-available/lastwar`:

```nginx
# Redirect HTTP to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name your-domain.com;
    
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }
    
    location / {
        return 301 https://$server_name$request_uri;
    }
}

# HTTPS server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name your-domain.com;
    
    # SSL certificates (will be configured by certbot)
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    
    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    ssl_stapling on;
    ssl_stapling_verify on;
    
    # Security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
    
    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req_status 429;
    
    # Logging
    access_log /var/log/nginx/lastwar-access.log;
    error_log /var/log/nginx/lastwar-error.log;
    
    # Proxy to Go application
    location / {
        limit_req zone=api_limit burst=20 nodelay;
        
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
    
    # Deny access to hidden files
    location ~ /\. {
        deny all;
        access_log off;
        log_not_found off;
    }
}
```

### 3. Enable Site and Get Certificate

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/lastwar /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl enable nginx

# Get Let's Encrypt certificate
sudo certbot --nginx -d your-domain.com

# Test auto-renewal
sudo certbot renew --dry-run
```

### 4. Start Services

```bash
sudo systemctl start lastwar
sudo systemctl restart nginx
```

---

## Additional Security Hardening

### 1. Keep System Updated

```bash
# Enable automatic security updates
sudo apt install -y unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades

# Manual updates
sudo apt update && sudo apt upgrade -y
```

### 2. Configure Fail2ban

```bash
sudo apt install -y fail2ban

# Create custom jail for nginx
sudo tee /etc/fail2ban/jail.local << 'EOF'
[nginx-http-auth]
enabled = true
port = http,https
logpath = /var/log/nginx/lastwar-error.log

[nginx-limit-req]
enabled = true
port = http,https
logpath = /var/log/nginx/lastwar-error.log
maxretry = 10
findtime = 600

[sshd]
enabled = true
port = ssh
logpath = /var/log/auth.log
maxretry = 5
bantime = 3600
EOF

sudo systemctl enable fail2ban
sudo systemctl start fail2ban
sudo fail2ban-client status
```

### 3. Secure SSH

Edit `/etc/ssh/sshd_config`:

```bash
# Disable root login
PermitRootLogin no

# Disable password authentication (use SSH keys)
PasswordAuthentication no
PubkeyAuthentication yes

# Change default port (optional)
Port 2222

# Limit users
AllowUsers yourusername
```

Then restart SSH:
```bash
sudo systemctl restart sshd
```

### 4. Database Backups

```bash
# Create backup script
sudo tee /usr/local/bin/backup-lastwar.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/var/backups/lastwar"
DB_PATH="/var/lib/lastwar/alliance.db"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR
sqlite3 $DB_PATH ".backup '$BACKUP_DIR/alliance_$DATE.db'"

# Keep only last 7 days
find $BACKUP_DIR -name "alliance_*.db" -mtime +7 -delete
EOF

sudo chmod +x /usr/local/bin/backup-lastwar.sh

# Add to crontab (daily at 2 AM)
echo "0 2 * * * /usr/local/bin/backup-lastwar.sh" | sudo crontab -
```

### 5. Application-Level Security Updates

Update `main.go` to use secure cookies:

```go
// In your login handler, set secure cookie flags
cookie := &http.Cookie{
    Name:     "session_token",
    Value:    token,
    HttpOnly: true,
    Secure:   true,  // Only send over HTTPS
    SameSite: http.SameSiteStrictMode,
    MaxAge:   86400,
    Path:     "/",
}
http.SetCookie(w, cookie)
```

### 6. Monitor Logs

```bash
# View application logs
sudo journalctl -u lastwar -f

# View Caddy logs
sudo journalctl -u caddy -f

# View Nginx logs (if using nginx)
sudo tail -f /var/log/nginx/lastwar-access.log
sudo tail -f /var/log/nginx/lastwar-error.log
```

### 7. System Resource Limits

Add to `/etc/security/limits.conf`:

```
lastwar soft nofile 65535
lastwar hard nofile 65535
lastwar soft nproc 4096
lastwar hard nproc 4096
```

---

## DNS Configuration

Before deploying, configure your domain's DNS:

```
A Record: your-domain.com → your.server.ip.address
```

Wait for DNS propagation (5-60 minutes) before running Certbot.

---

## Monitoring (Optional)

### Simple Monitoring Script

```bash
sudo tee /usr/local/bin/monitor-lastwar.sh << 'EOF'
#!/bin/bash
if ! systemctl is-active --quiet lastwar; then
    systemctl start lastwar
    echo "Last War service was down, restarted at $(date)" >> /var/log/lastwar-monitor.log
fi
EOF

sudo chmod +x /usr/local/bin/monitor-lastwar.sh
echo "*/5 * * * * /usr/local/bin/monitor-lastwar.sh" | sudo crontab -
```

---

## Quick Troubleshooting

```bash
# Check if application is running
sudo systemctl status lastwar

# Check application logs
sudo journalctl -u lastwar -n 50

# Check if port 8080 is listening
sudo ss -tlnp | grep 8080

# Test reverse proxy
curl http://localhost:8080

# Check SSL certificate
echo | openssl s_client -servername your-domain.com -connect your-domain.com:443 2>/dev/null | openssl x509 -noout -dates

# Restart everything
sudo systemctl restart lastwar
sudo systemctl restart caddy  # or nginx
```

---

## Performance Tuning

For high traffic, consider:

1. **Connection pooling**: Already handled by SQLite driver
2. **Response caching**: Add caching headers in Caddy/Nginx
3. **Database optimization**: Regular VACUUM and ANALYZE
4. **Load balancing**: Run multiple instances behind load balancer

---

## Security Checklist

- [x] Application runs as non-root user
- [x] Firewall configured (UFW)
- [x] HTTPS with Let's Encrypt
- [x] Security headers configured
- [x] Rate limiting enabled
- [x] Fail2ban configured
- [x] SSH hardened
- [x] Automatic security updates
- [x] Database backups configured
- [x] Logs monitored
- [x] Secure session cookies

---

## Update Procedure

```bash
# Stop service
sudo systemctl stop lastwar

# Backup database
sudo cp /var/lib/lastwar/alliance.db /var/lib/lastwar/alliance.db.backup

# Update code
cd /opt/lastwar
git pull  # or upload new files

# Rebuild
go build -o alliance-manager main.go

# Restart service
sudo systemctl start lastwar
sudo systemctl status lastwar
```
