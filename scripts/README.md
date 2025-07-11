# Router Sync Linux Installation

This directory contains the installation scripts and systemd service file for Router Sync on Linux systems.

## Files

- `install.sh` - Main installation script
- `router-sync.service` - Systemd service unit file
- `README.md` - This file

## Installation

### Prerequisites

- Linux system with systemd
- Root or sudo access
- Supported architecture (amd64 or arm64)

### Quick Installation

1. Extract the tar.gz file:
   ```bash
   tar -xzf router-sync-v<VERSION>-linux-<ARCH>.tar.gz
   cd router-sync-v<VERSION>-linux-<ARCH>
   ```

2. Run the installation script:
   ```bash
   sudo ./install.sh
   ```

The script will:
- Detect your system architecture
- Create a system user `router-sync`
- Install the binary to `/usr/local/bin/`
- Create configuration directory at `/etc/router-sync/`
- Install and enable the systemd service
- Start the service automatically

### Manual Installation

If you prefer to install manually:

1. Copy the binary to `/usr/local/bin/`:
   ```bash
   sudo cp router-sync-linux-<ARCH> /usr/local/bin/router-sync
   sudo chmod 755 /usr/local/bin/router-sync
   ```

2. Create the service user:
   ```bash
   sudo useradd --system --no-create-home --shell /bin/false router-sync
   ```

3. Create directories:
   ```bash
   sudo mkdir -p /etc/router-sync
   sudo mkdir -p /var/log/router-sync
   sudo mkdir -p /var/lib/router-sync
   sudo chown router-sync:router-sync /var/log/router-sync
   sudo chown router-sync:router-sync /var/lib/router-sync
   ```

4. Copy the systemd service file:
   ```bash
   sudo cp router-sync.service /etc/systemd/system/
   sudo systemctl daemon-reload
   ```

5. Create configuration file:
   ```bash
   sudo cp config.yaml /etc/router-sync/
   sudo chown router-sync:router-sync /etc/router-sync/config.yaml
   ```

6. Enable and start the service:
   ```bash
   sudo systemctl enable router-sync
   sudo systemctl start router-sync
   ```

## Configuration

The service uses the configuration file at `/etc/router-sync/config.yaml`. Edit this file to customize:

- NATS connection settings
- API server address
- Log level
- Sync interval

After making changes, restart the service:
```bash
sudo systemctl restart router-sync
```

## Service Management

### Check service status:
```bash
sudo systemctl status router-sync
```

### View logs:
```bash
sudo journalctl -u router-sync -f
```

### Start/Stop/Restart:
```bash
sudo systemctl start router-sync
sudo systemctl stop router-sync
sudo systemctl restart router-sync
```

### Enable/Disable auto-start:
```bash
sudo systemctl enable router-sync
sudo systemctl disable router-sync
```

## Uninstallation

To completely remove Router Sync:

```bash
# Stop and disable the service
sudo systemctl stop router-sync
sudo systemctl disable router-sync

# Remove files
sudo rm -f /etc/systemd/system/router-sync.service
sudo rm -f /usr/local/bin/router-sync
sudo rm -rf /etc/router-sync
sudo rm -rf /var/log/router-sync
sudo rm -rf /var/lib/router-sync

# Remove service user
sudo userdel router-sync 2>/dev/null || true
sudo groupdel router-sync 2>/dev/null || true

# Reload systemd
sudo systemctl daemon-reload
```

## Troubleshooting

### Service won't start
Check the service status and logs:
```bash
sudo systemctl status router-sync
sudo journalctl -u router-sync -n 50
```

### Permission issues
Ensure the service user has proper permissions:
```bash
sudo chown router-sync:router-sync /var/log/router-sync
sudo chown router-sync:router-sync /var/lib/router-sync
sudo chown router-sync:router-sync /etc/router-sync/config.yaml
```

### Configuration issues
Validate your configuration file:
```bash
sudo -u router-sync /usr/local/bin/router-sync -config /etc/router-sync/config.yaml -validate
```

## Security Notes

- The service runs as a dedicated system user `router-sync`
- Systemd security features are enabled (NoNewPrivileges, PrivateTmp, etc.)
- The service has limited file system access through ReadWritePaths
- Configuration files are owned by the service user

## Support

For issues and questions:
- Check the logs: `sudo journalctl -u router-sync`
- Review the configuration: `/etc/router-sync/config.yaml`
- Visit the project repository for documentation and issues 