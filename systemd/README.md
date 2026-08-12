<!-- file: systemd/README.md -->
<!-- version: 1.1.0 -->
<!-- last-edited: 2026-08-12 -->
<!-- guid: 789e1234-e89b-12d3-a456-426614174002 -->

# Systemd Service Installation Guide

This guide provides instructions for running Subtitle Manager as a systemd
service on Linux systems.

## Prerequisites

- Linux system with systemd
- Go 1.21+ or precompiled binary
- ffmpeg installed (`sudo apt install ffmpeg` on Ubuntu/Debian)

## Installation Steps

### Quick Installation (Recommended)

Use the automated installation script for easy setup:

```bash
# From the subtitle-manager repository root
sudo ./systemd/install.sh
```

The script will:

- Create system user and directories
- Install the binary and service files
- Configure systemd service
- Display next steps for configuration

### Manual Installation

For custom setups or troubleshooting, follow these manual steps:

#### 1. Create System User

#### 1. Create System User

Create a dedicated user for running the service:

```bash
sudo useradd --system --home-dir /var/lib/subtitle-manager --create-home --shell /bin/false subtitle-manager
```

#### 2. Create Required Directories

#### 2. Create Required Directories

```bash
sudo mkdir -p /etc/subtitle-manager
sudo mkdir -p /var/lib/subtitle-manager/db
sudo mkdir -p /var/log/subtitle-manager
sudo mkdir -p /opt/subtitle-manager

# Set ownership
sudo chown -R subtitle-manager:subtitle-manager /var/lib/subtitle-manager
sudo chown -R subtitle-manager:subtitle-manager /var/log/subtitle-manager
sudo chown subtitle-manager:subtitle-manager /opt/subtitle-manager
```

#### 3. Install Binary

#### 3. Install Binary

##### Option A: From Release

Download the latest release binary:

```bash
# Download for Linux x64
curl -L https://github.com/jdfalk/subtitle-manager/releases/latest/download/subtitle-manager-linux-amd64 -o subtitle-manager
sudo mv subtitle-manager /usr/local/bin/subtitle-manager
sudo chmod +x /usr/local/bin/subtitle-manager
```

##### Option B: Build from Source

```bash
git clone https://github.com/jdfalk/subtitle-manager.git
cd subtitle-manager

# One build supports both the SQLite and PebbleDB backends
CGO_ENABLED=0 go build -o subtitle-manager .

sudo cp bin/subtitle-manager /usr/local/bin/subtitle-manager
sudo chmod +x /usr/local/bin/subtitle-manager
```

**Note**: For production deployments, building with SQLite support is
recommended for better database compatibility. The default systemd configuration
supports both SQLite and PebbleDB backends.

#### 4. Install Service Files

#### 4. Install Service Files

```bash
# Copy systemd service file
sudo cp systemd/subtitle-manager.service /etc/systemd/system/

# Copy environment file template
sudo cp systemd/subtitle-manager.env /etc/subtitle-manager/

# Set permissions
sudo chmod 644 /etc/systemd/system/subtitle-manager.service
sudo chmod 640 /etc/subtitle-manager/subtitle-manager.env
sudo chown root:subtitle-manager /etc/subtitle-manager/subtitle-manager.env
```

#### 5. Configure Environment

#### 5. Configure Environment

Edit the environment file to match your setup:

```bash
sudo nano /etc/subtitle-manager/subtitle-manager.env
```

Key settings to configure:

- `SM_LOG_LEVEL`: Set logging level (debug, info, warn, error)
- `SM_DB_BACKEND`: Choose database backend (sqlite, pebble, postgres)
- API keys for translation and subtitle providers
- Media and subtitle paths if using file scanning features

#### 6. Create Configuration File (Optional)

#### 6. Create Configuration File (Optional)

Create a YAML configuration file for advanced settings:

```bash
sudo nano /etc/subtitle-manager/subtitle-manager.yaml
```

Example configuration:

```yaml
log-level: info
db_backend: sqlite
db_path: /var/lib/subtitle-manager/db
log_file: /var/log/subtitle-manager/subtitle-manager.log

# Provider settings
providers:
  opensubtitles:
    api_key: 'your_api_key_here'

# Translation settings
translator: google
translator_api_keys:
  google: 'your_google_api_key'
```

#### 7. Enable and Start Service

#### 7. Enable and Start Service

```bash
# Reload systemd to recognize the new service
sudo systemctl daemon-reload

# Enable service to start on boot
sudo systemctl enable subtitle-manager

# Start the service
sudo systemctl start subtitle-manager

# Check service status
sudo systemctl status subtitle-manager
```

## Service Management

### Common Commands

```bash
# Start service
sudo systemctl start subtitle-manager

# Stop service
sudo systemctl stop subtitle-manager

# Restart service
sudo systemctl restart subtitle-manager

# Reload configuration (if supported by app)
sudo systemctl reload subtitle-manager

# Enable service to start on boot
sudo systemctl enable subtitle-manager

# Disable service from starting on boot
sudo systemctl disable subtitle-manager

# Check service status
sudo systemctl status subtitle-manager

# View service logs
sudo journalctl -u subtitle-manager -f

# View logs since last boot
sudo journalctl -u subtitle-manager -b

# View logs for specific time period
sudo journalctl -u subtitle-manager --since "2024-01-01 00:00:00"
```

### Log Files

Logs are written to both systemd journal and file (if configured):

- **Systemd journal**: `sudo journalctl -u subtitle-manager`
- **Log file**: `/var/log/subtitle-manager/subtitle-manager.log` (if
  `SM_LOG_FILE` is set)

## Accessing the Web Interface

Once the service is running, access the web interface at:

- **Local**: http://localhost:8080
- **Network**: http://your-server-ip:8080

## Firewall Configuration

If using a firewall, allow access to port 8080:

```bash
# UFW (Ubuntu)
sudo ufw allow 8080/tcp

# iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

## Advanced Configuration

### Custom Port

To run on a different port, edit the service file:

```bash
sudo systemctl edit subtitle-manager
```

Add the following override:

```ini
[Service]
ExecStart=
ExecStart=/usr/local/bin/subtitle-manager web --addr :9090
Environment=SM_WEB_ADDR=:9090
```

### PostgreSQL Database

For PostgreSQL backend, ensure PostgreSQL is installed and create a database:

```sql
CREATE DATABASE subtitle_manager;
CREATE USER subtitle_manager WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE subtitle_manager TO subtitle_manager;
```

Update environment file:

```bash
SM_DB_BACKEND=postgres
SM_POSTGRES_DSN=postgres://subtitle_manager:your_password@localhost:5432/subtitle_manager?sslmode=disable
```

### SSL/TLS Configuration

For HTTPS, use a reverse proxy like nginx:

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Troubleshooting

### Service Won't Start

1. Check service status: `sudo systemctl status subtitle-manager`
2. Check logs: `sudo journalctl -u subtitle-manager`
3. Verify binary permissions: `ls -la /usr/local/bin/subtitle-manager`
4. Check configuration:
   `sudo -u subtitle-manager /usr/local/bin/subtitle-manager --help`

### Permission Issues

Ensure the service user has access to required directories:

```bash
sudo chown -R subtitle-manager:subtitle-manager /var/lib/subtitle-manager
sudo chown -R subtitle-manager:subtitle-manager /var/log/subtitle-manager
```

### Database Issues

For SQLite database permission issues:

```bash
sudo chown subtitle-manager:subtitle-manager /var/lib/subtitle-manager/db/
sudo chmod 755 /var/lib/subtitle-manager/db/
```

### Web Interface Not Accessible

1. Check if service is listening: `sudo netstat -tlnp | grep 8080`
2. Verify firewall settings
3. Check if another service is using port 8080: `sudo lsof -i :8080`

## Security Considerations

- The service runs as a non-privileged user (`subtitle-manager`)
- Sensitive configuration is stored in `/etc/subtitle-manager/` with restricted
  permissions
- Database and logs are stored in `/var/lib/subtitle-manager/` and
  `/var/log/subtitle-manager/`
- The service includes security hardening options in the systemd unit file
- Consider using a reverse proxy for SSL termination and additional security
  features

## Upgrading

To upgrade Subtitle Manager:

1. Stop the service: `sudo systemctl stop subtitle-manager`
2. Replace the binary:
   `sudo cp new-subtitle-manager /usr/local/bin/subtitle-manager`
3. Start the service: `sudo systemctl start subtitle-manager`

For major version upgrades, check the changelog for any required configuration
changes.

## Uninstalling

To completely remove the service:

```bash
# Stop and disable service
sudo systemctl stop subtitle-manager
sudo systemctl disable subtitle-manager

# Remove service files
sudo rm /etc/systemd/system/subtitle-manager.service
sudo rm -rf /etc/subtitle-manager/

# Remove binary
sudo rm /usr/local/bin/subtitle-manager

# Remove user and data (optional)
sudo userdel subtitle-manager
sudo rm -rf /var/lib/subtitle-manager/
sudo rm -rf /var/log/subtitle-manager/

# Reload systemd
sudo systemctl daemon-reload
```
