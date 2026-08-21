# Quick Production Setup Guide

## Local Development & Testing

Run the app locally with Docker and validate it with the included Playwright test suite. No Go, GCC, or Tesseract installation required.

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) with Compose v2
- [Node.js](https://nodejs.org/) (for Playwright)

### 1. Build and Start

```bash
docker compose -f docker-compose.dev.yml up -d --build
```

App is now at **http://localhost:8080**. Default credentials: `admin` / `admin123`.

### 2. Set Up Playwright

```bash
cd playwright-tests
npm install
npx playwright install chromium
```

### 3. Run the Tests

```bash
npx playwright test        # headless (CI-friendly)
npm test                   # same thing via npm script
npx playwright test --headed   # show browser window
```

The test runner logs in automatically via `global-setup.js` — no manual login needed.

### 4. View the HTML Report

```bash
npm run report
# opens playwright-tests/report/index.html
```

Screenshots from each run are saved to `playwright-tests/screenshots/`.

### Iterate: Rebuild and Retest

```bash
docker compose -f docker-compose.dev.yml up -d --build   # rebuild image after code changes
cd playwright-tests && npx playwright test
```

### Test Files

| File | Coverage |
|---|---|
| `tests/app.spec.js` | Login, members, train schedule, awards, navigation |
| `tests/vs-ocr.spec.js` | VS Points OCR upload (auto-skips if screenshot not present) |

---

## Option 1: Docker / Podman (Fastest — Recommended)

No Go, GCC, or Tesseract needed on the host.

### 1. Generate a Session Key
```bash
openssl rand -hex 32
```

### 2. Set the Session Key
Create a `.env` file next to `docker-compose.yml` (Compose reads it automatically):

```bash
echo "SESSION_KEY=your-64-char-hex-key-here" > .env
chmod 600 .env
```

### 3. Start
```bash
# Docker
docker compose up -d

# Podman
podman-compose up -d
```

The app is running at `http://localhost:8080`.  
The database is stored in `./data/alliance.db` on the host.

### 4. Add HTTPS (Caddy)
```bash
sudo apt install -y caddy
echo 'your-domain.com { reverse_proxy localhost:8080 }' | sudo tee /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

### Useful Commands
```bash
docker compose logs -f                                   # Tail logs
docker compose restart                                   # Restart
docker compose pull && docker compose up -d              # Pull latest published image
docker compose -f docker-compose.dev.yml up -d --build   # Rebuild from local source
docker compose down                                      # Stop and remove containers
```

See [DEPLOYMENT.md](DEPLOYMENT.md) for full Docker production setup.

---

## Option 2: One-Command Bare-Metal Installation (Debian/Ubuntu)

```bash
chmod +x install.sh
sudo ./install.sh
```

The script will:
- ✅ Install Go and build dependencies (gcc, g++, tesseract)
- ✅ Create system user and directories
- ✅ Build the application
- ✅ Generate secure session key
- ✅ Configure systemd service
- ✅ Setup firewall (UFW)
- ✅ Install Caddy or Nginx
- ✅ Configure Let's Encrypt SSL
- ✅ Setup fail2ban
- ✅ Configure daily backups

## Manual Quick Start (Bare-Metal)

### 1. Prerequisites
```bash
# Domain pointing to your server IP
# Ports 80 and 443 open in firewall
```

### 2. Generate Session Key
```bash
openssl rand -hex 32
```

### 3. Create Environment File
```bash
cat > .env << EOF
DATABASE_PATH=/var/lib/lastwar/alliance.db
SESSION_KEY=your-generated-key-here
PRODUCTION=true
HTTPS=true
EOF
```

### 4. Choose Your Reverse Proxy

#### Option A: Caddy (Recommended - Automatic HTTPS)
```bash
# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install caddy

# Configure
echo "your-domain.com {
    reverse_proxy localhost:8080
}" | sudo tee /etc/caddy/Caddyfile

sudo systemctl restart caddy
```

#### Option B: Nginx + Certbot
```bash
# Install
sudo apt install -y nginx certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d your-domain.com
```

### 5. Start Application
```bash
# Install build deps if not already done
sudo apt install -y gcc g++ build-essential tesseract-ocr tesseract-ocr-eng libtesseract-dev libleptonica-dev

# Build
go build -o alliance-manager main.go

# Run with environment
export $(grep -v '^#' .env | xargs)
./alliance-manager
```

Or use the systemd service (see [DEPLOYMENT.md](DEPLOYMENT.md)).

## Essential Commands

### Docker / Podman
```bash
docker compose up -d              # Start (detached)
docker compose down               # Stop
docker compose logs -f            # Tail logs
docker compose restart            # Restart
docker compose up -d --build      # Rebuild after code change
```

### Systemd Service (Bare-Metal)
```bash
sudo systemctl status lastwar      # Check status
sudo systemctl start lastwar       # Start service
sudo systemctl stop lastwar        # Stop service
sudo systemctl restart lastwar     # Restart service
sudo journalctl -u lastwar -f      # View logs
```

### SSL Certificate
```bash
# Caddy (automatic renewal)
sudo systemctl status caddy

# Nginx (manual renewal test)
sudo certbot renew --dry-run
```

### Backups
```bash
# Docker: copy the database from the host-mounted volume
cp ./data/alliance.db ./data/alliance_$(date +%Y%m%d_%H%M%S).db

# Bare-metal: use the backup script installed by install.sh
sudo /usr/local/bin/backup-lastwar.sh

# List backups
ls -lh /var/backups/lastwar/

# Restore from backup (bare-metal)
sudo cp /var/backups/lastwar/alliance_YYYYMMDD_HHMMSS.db /var/lib/lastwar/alliance.db
sudo chown lastwar:lastwar /var/lib/lastwar/alliance.db
sudo systemctl restart lastwar

# Restore from backup (Docker)
cp ./data/alliance_YYYYMMDD_HHMMSS.db ./data/alliance.db
docker compose restart
```

### Security Checks
```bash
# Check firewall
sudo ufw status

# Check fail2ban
sudo fail2ban-client status

# Check banned IPs
sudo fail2ban-client status sshd

# Unban IP
sudo fail2ban-client set sshd unbanip 123.123.123.123

# Check SSL grade
curl https://www.ssllabs.com/ssltest/analyze.html?d=your-domain.com
```

### Updates
```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Update application (Docker)
cd /opt/lastwar
git pull
docker compose up -d --build

# Update application (bare-metal)
cd /opt/lastwar
git pull
go build -o alliance-manager main.go
sudo systemctl restart lastwar
```

## Troubleshooting

### Service won't start
```bash
# Check logs
sudo journalctl -u lastwar -n 100

# Check if port is available
sudo ss -tlnp | grep 8080

# Check permissions
sudo ls -la /var/lib/lastwar/
```

### SSL certificate issues
```bash
# Caddy: Check logs
sudo journalctl -u caddy -n 50

# Nginx: Test configuration
sudo nginx -t

# Check DNS
dig your-domain.com
```

### Database locked
```bash
# Check for open connections
sudo lsof /var/lib/lastwar/alliance.db

# Restart service
sudo systemctl restart lastwar
```

### High memory usage
```bash
# Check memory
free -h

# Check Go memory
sudo systemctl status lastwar
```

## Security Best Practices

- ✅ Change default admin password immediately
- ✅ Use strong, unique SESSION_KEY
- ✅ Keep system updated (`sudo apt update && sudo apt upgrade`)
- ✅ Monitor logs regularly
- ✅ Test backups periodically
- ✅ Use SSH keys instead of passwords
- ✅ Enable 2FA for SSH (optional but recommended)
- ✅ Monitor fail2ban bans
- ✅ Use non-standard SSH port (optional)
- ✅ Implement rate limiting (already configured)

## Performance Optimization

### For high traffic:
```bash
# Increase system limits
echo "lastwar soft nofile 65535" | sudo tee -a /etc/security/limits.conf
echo "lastwar hard nofile 65535" | sudo tee -a /etc/security/limits.conf

# Enable kernel tweaks
sudo sysctl -w net.core.somaxconn=65535
sudo sysctl -w net.ipv4.tcp_max_syn_backlog=8192
```

### Enable compression in Caddy:
Already configured in provided Caddyfile

### Database optimization:
```bash
# Run VACUUM monthly
echo "0 3 1 * * sqlite3 /var/lib/lastwar/alliance.db 'VACUUM;'" | sudo crontab -
```

## Monitoring (Simple)

### Create monitoring script:
```bash
#!/bin/bash
# /usr/local/bin/check-lastwar.sh

if ! curl -f http://localhost:8080 > /dev/null 2>&1; then
    echo "Service down at $(date)" >> /var/log/lastwar-monitor.log
    systemctl restart lastwar
fi
```

### Add to crontab (every 5 minutes):
```bash
*/5 * * * * /usr/local/bin/check-lastwar.sh
```

## Need Help?

1. Check [DEPLOYMENT.md](DEPLOYMENT.md) for the detailed guide
2. Docker logs: `docker compose logs -f`
3. Systemd logs: `sudo journalctl -u lastwar -f`
4. Check application status: `sudo systemctl status lastwar` or `docker compose ps`
5. Verify reverse proxy: `sudo systemctl status caddy` or `sudo systemctl status nginx`
6. Test database (bare-metal): `sqlite3 /var/lib/lastwar/alliance.db ".tables"`
7. Test database (Docker): `docker compose exec lastwar sh -c 'sqlite3 /data/alliance.db ".tables"'`

## Quick Health Check

### Docker
```bash
docker compose ps
docker compose logs --tail=20
curl -sf http://localhost:8080 && echo "OK" || echo "DOWN"
```

### Bare-Metal
Run this to verify everything is working:

```bash
echo "=== Service Status ==="
sudo systemctl is-active lastwar caddy fail2ban

echo -e "\n=== Port Check ==="
sudo ss -tlnp | grep -E '(8080|80|443)'

echo -e "\n=== SSL Certificate ==="
echo | openssl s_client -servername your-domain.com -connect your-domain.com:443 2>/dev/null | openssl x509 -noout -dates

echo -e "\n=== Database ==="
sudo sqlite3 /var/lib/lastwar/alliance.db "SELECT COUNT(*) FROM members;"

echo -e "\n=== Recent Logs ==="
sudo journalctl -u lastwar --since "5 minutes ago" --no-pager
```
