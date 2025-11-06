# aistack Operations Playbook

> Operational procedures, troubleshooting guides, and common tasks for aistack administrators.

## Table of Contents

1. [Service Management](#service-management)
2. [Troubleshooting](#troubleshooting)
3. [Update & Rollback](#update--rollback)
4. [Backup & Recovery](#backup--recovery)
5. [Performance Tuning](#performance-tuning)
6. [Common Error Patterns](#common-error-patterns)

---

## Service Management

### Check Service Status

```bash
# View all services
aistack status

# Detailed health report
aistack health --save

# Check specific service logs
aistack logs ollama
aistack logs openwebui 100  # Last 100 lines
```

### Start/Stop Services

```bash
# Start individual service
aistack start ollama

# Stop individual service
aistack stop ollama

# Restart by stopping then starting
aistack stop ollama && aistack start ollama
```

### Service Installation

```bash
# Install with profile
aistack install --profile standard-gpu  # Ollama + OpenWebUI + LocalAI
aistack install --profile minimal       # Ollama only

# Install individual service
aistack install ollama
aistack install openwebui
aistack install localai
```

### Service Removal

```bash
# Remove service (keeps data volumes)
aistack remove ollama

# Remove service and purge data
aistack remove ollama --purge

# Complete system purge (double confirmation required)
aistack purge --all
aistack purge --all --remove-configs  # Also removes /etc/aistack
```

---

## Troubleshooting

### Problem: Service Shows Red Status

**Symptoms**: `aistack status` shows service in "red" health state

**Diagnosis**:
```bash
# Check service logs
aistack logs <service> 50

# Check container status
docker ps -a | grep aistack

# Check docker compose status
cd /usr/share/aistack/compose  # Or wherever compose files are located
docker compose -f <service>.yaml ps
```

**Resolution**:
```bash
# Attempt automatic repair
aistack repair <service>

# Or manual restart
aistack stop <service>
aistack start <service>

# If persistent, check for port conflicts
sudo netstat -tulpn | grep <port>  # e.g., 11434 for Ollama
```

### Problem: GPU Not Detected

**Symptoms**: `aistack gpu-check` reports no GPU or NVIDIA stack issues

**Diagnosis**:
```bash
# Verify NVIDIA driver
nvidia-smi

# Check Docker GPU runtime
docker run --rm --gpus all nvidia/cuda:12.0.0-base-ubuntu22.04 nvidia-smi

# Check NVML library
ldconfig -p | grep nvidia-ml
```

**Resolution**:
```bash
# Install NVIDIA drivers if missing
sudo ubuntu-drivers devices
sudo ubuntu-drivers autoinstall

# Install nvidia-container-toolkit
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
  sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
  sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit

# Restart Docker
sudo systemctl restart docker

# Re-check
aistack gpu-check
```

### Problem: Port Already in Use

**Symptoms**: Service fails to start with "port already allocated" error

**Diagnosis**:
```bash
# Check what's using the port
sudo netstat -tulpn | grep <port>
# OR
sudo lsof -i :<port>

# Common ports:
# - Ollama: 11434
# - Open WebUI: 3000
# - LocalAI: 8080
```

**Resolution**:
```bash
# Option 1: Stop conflicting service
sudo systemctl stop <conflicting-service>

# Option 2: Change aistack port in compose file
# Edit /usr/share/aistack/compose/<service>.yaml
# Change ports section: "3001:3000" (external:internal)
```

### Problem: Service Update Failed

**Symptoms**: `aistack update <service>` returns error

**Diagnosis**:
```bash
# Check update plan
cat /var/lib/aistack/<service>_update_plan.json

# Check if rollback occurred
aistack logs <service> 100 | grep -i rollback

# Verify image availability
docker pull <image-name>
```

**Resolution**:
```bash
# If update policy is blocking:
aistack versions  # Check if mode=pinned
# Edit /etc/aistack/config.yaml: set updates.mode to "rolling"

# Force recreate service
aistack stop <service>
aistack remove <service>  # Keeps data
aistack install <service>
```

### Problem: Out of Disk Space

**Symptoms**: Services fail with I/O errors, `df -h` shows full disk

**Diagnosis**:
```bash
# Check disk usage
df -h

# Check Docker disk usage
docker system df

# Check aistack data
du -sh /var/lib/aistack/volumes/*
```

**Resolution**:
```bash
# Clean Docker system (removes stopped containers, dangling images)
docker system prune -a

# Evict oldest models
aistack models evict-oldest ollama
aistack models evict-oldest localai

# Remove unused services
aistack remove <service> --purge
```

---

## Update & Rollback

### Update Single Service

```bash
# Update with automatic rollback on health failure
aistack update ollama

# Check update status
aistack logs ollama 50
```

### Update All Services

```bash
# Sequential update: LocalAI → Ollama → Open WebUI
aistack update-all

# Each service is updated independently
# Failure in one service does not affect others
```

### Version Pinning

```bash
# Check current update policy
aistack versions

# Create version lock file
sudo nano /etc/aistack/versions.lock

# Example content:
# ollama:ollama/ollama@sha256:abc123...
# openwebui:ghcr.io/open-webui/open-webui:v0.1.0
# localai:quay.io/go-skynet/local-ai:v2.8.0

# Set update mode to pinned
sudo nano /etc/aistack/config.yaml
# Set: updates.mode: pinned

# Verify
aistack versions
```

### Manual Rollback

```bash
# Stop service
aistack stop <service>

# Remove current container (keeps data)
docker rm aistack-<service>

# Update versions.lock to previous version
sudo nano /etc/aistack/versions.lock

# Start service (will use locked version)
aistack start <service>

# Verify
aistack status
aistack health
```

---

## Backup & Recovery

### Backup Service Data

```bash
# Stop service
aistack stop <service>

# Backup volume data
docker run --rm \
  -v <volume-name>:/data \
  -v $(pwd):/backup \
  ubuntu tar czf /backup/<service>-backup-$(date +%Y%m%d).tar.gz /data

# Example for Ollama:
docker run --rm \
  -v ollama_data:/data \
  -v $(pwd):/backup \
  ubuntu tar czf /backup/ollama-backup-$(date +%Y%m%d).tar.gz /data

# Start service
aistack start <service>
```

### Restore Service Data

```bash
# Stop service
aistack stop <service>

# Remove existing volume (CAUTION: Data loss!)
docker volume rm <volume-name>

# Restore from backup
docker run --rm \
  -v <volume-name>:/data \
  -v $(pwd):/backup \
  ubuntu tar xzf /backup/<service>-backup-<date>.tar.gz -C /

# Start service
aistack start <service>
```

### Backup Configuration

```bash
# Backup all aistack configuration
sudo tar czf aistack-config-backup-$(date +%Y%m%d).tar.gz \
  /etc/aistack \
  /var/lib/aistack/*.json \
  /var/lib/aistack/wol_config.json

# Restore configuration
sudo tar xzf aistack-config-backup-<date>.tar.gz -C /
```

---

## Performance Tuning

### GPU Utilization

```bash
# Monitor GPU usage
watch -n 1 nvidia-smi

# Check GPU lock status
aistack status | grep -A5 "GPU Lock"

# Manually unlock if stuck
aistack gpu-unlock
```

### Idle Detection Tuning

```bash
# Edit configuration
sudo nano /etc/aistack/config.yaml

# Adjust thresholds:
idle:
  cpu_idle_threshold: 10      # CPU below 10% = idle
  gpu_idle_threshold: 5       # GPU below 5% = idle
  window_seconds: 300         # 5-minute sliding window
  idle_timeout_seconds: 1800  # Suspend after 30 min idle

# Test idle detection
aistack idle-check
aistack idle-check --ignore-inhibitors  # Force check
```

### Model Cache Management

```bash
# List models
aistack models list ollama
aistack models list localai

# Show cache statistics
aistack models stats ollama
aistack models stats localai

# Delete unused models
aistack models delete ollama <model-name>

# Evict oldest model
aistack models evict-oldest ollama
```

---

## Common Error Patterns

### Pattern: "Cannot connect to Docker daemon"

**Error**: `Cannot connect to the Docker daemon at unix:///var/run/docker.sock`

**Cause**: User not in docker group, or Docker not running

**Fix**:
```bash
# Add user to docker group
sudo usermod -aG docker $USER
# Logout and login again

# Start Docker if not running
sudo systemctl start docker
sudo systemctl enable docker
```

### Pattern: "Network aistack-net not found"

**Error**: Service startup fails with network error

**Cause**: Docker network was manually deleted

**Fix**:
```bash
# Recreate network
docker network create aistack-net

# Or reinstall service
aistack stop <service>
aistack install <service>
```

### Pattern: "Volume in use" during purge

**Error**: Cannot remove volume, still in use

**Cause**: Container not fully stopped

**Fix**:
```bash
# Force remove all aistack containers
docker ps -a | grep aistack | awk '{print $1}' | xargs docker rm -f

# Retry removal
aistack remove <service> --purge
```

### Pattern: Health check timeout

**Error**: Service stuck in "yellow" or "red", logs show timeout

**Cause**: Service taking longer than expected to initialize

**Fix**:
```bash
# Wait for service to fully start (especially on first run)
# Services may download models on first startup

# Check progress
aistack logs <service> -f  # Follow logs

# For Ollama first startup:
# Wait for "Ollama server started" message
# May take 1-5 minutes depending on network
```

---

## Emergency Procedures

### Complete System Reset

```bash
# ⚠️ WARNING: This removes ALL aistack services and data

# Purge everything
aistack purge --all --remove-configs --yes

# Clean Docker system
docker system prune -a -f --volumes

# Verify clean slate
aistack status  # Should show "No services installed"
```

### Recovery from Corrupted State

```bash
# Stop agent if running
sudo systemctl stop aistack-agent

# Reset state directory
sudo rm -rf /var/lib/aistack/*

# Reset configuration (if needed)
sudo rm -rf /etc/aistack
sudo mkdir -p /etc/aistack

# Reinstall from scratch
aistack install --profile standard-gpu
```

---

## Monitoring & Logging

### View Logs

```bash
# Service logs
aistack logs ollama
aistack logs openwebui
aistack logs localai

# System logs (if agent is running)
sudo journalctl -u aistack-agent -f

# Metrics logs
tail -f /var/log/aistack/metrics.log | jq .
```

### Create Diagnostic Package

```bash
# Generate diagnostic ZIP (secrets redacted)
aistack diag

# With custom output path
aistack diag --output /tmp/aistack-diag.zip

# Exclude logs or config
aistack diag --no-logs
aistack diag --no-config
```

---

## Best Practices

1. **Always use health checks**: Run `aistack health` after changes
2. **Monitor disk space**: Set up alerts for disk usage >80%
3. **Regular backups**: Schedule weekly backups of data volumes
4. **Version locking in production**: Use `versions.lock` for stability
5. **Test updates**: Update test environment before production
6. **Keep logs**: Retain diagnostic packages for 30 days
7. **Document changes**: Track configuration changes in version control

---

For power management and Wake-on-LAN procedures, see [POWER_AND_WOL.md](POWER_AND_WOL.md).

For development and contribution guidelines, see [CONTRIBUTING.md](../CONTRIBUTING.md).
