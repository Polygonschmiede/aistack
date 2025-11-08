# aistack Troubleshooting Guide

## RAPL Permission Denied (CPU Power Monitoring)

### Symptoms
Log shows repeated warnings:
```
{"level":"warn","type":"cpu.rapl.read.failed","message":"Failed to read RAPL","payload":{"error":"failed to read RAPL: open /sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj: permission denied"}}
```

### Cause
The udev rule for RAPL permissions may not have been applied correctly, or you installed aistack before the fix was added.

### Fix Option 1: Update and Reinstall (Recommended)

```bash
# Pull latest changes
cd ~/aistack
git pull

# Reinstall (this will redeploy all configs)
sudo ./install.sh
```

### Fix Option 2: Manual Permission Fix (Immediate)

```bash
# Give read permissions to all RAPL energy files
sudo chmod 644 /sys/class/powercap/intel-rapl/*/energy_uj
sudo chmod 644 /sys/class/powercap/intel-rapl/*/*/energy_uj

# Restart the agent to pick up changes
sudo systemctl restart aistack-agent
```

**Note**: This fix is temporary and will be lost after reboot.

### Fix Option 3: Deploy tmpfiles Configuration (Persistent)

```bash
# Copy the tmpfiles configuration
sudo cp ~/aistack/assets/tmpfiles.d/aistack-rapl.conf /etc/tmpfiles.d/

# Apply it immediately
sudo systemd-tmpfiles --create /etc/tmpfiles.d/aistack-rapl.conf

# Verify permissions
ls -la /sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj

# Restart the agent
sudo systemctl restart aistack-agent
```

### Verification

Check logs to confirm RAPL is working:
```bash
sudo journalctl -u aistack-agent -f | grep rapl
```

You should see:
```
{"level":"info","type":"cpu.rapl.detected","message":"RAPL power monitoring available"}
```

Instead of:
```
{"level":"warn","type":"cpu.rapl.read.failed"}
```

---

## Idle State Constantly Resetting

### Symptoms
- Idle timer shows "<1s" and never increases
- TUI shows "Idle for: <1s" even when system is truly idle
- Gating reasons include "below_timeout" but timer never progresses

### Possible Causes

#### 1. Multiple aistack Instances Running

If you run both the TUI and the agent simultaneously, they may interfere with each other:

```bash
# Check if multiple instances are running
ps aux | grep aistack

# You might see both:
# - /usr/local/bin/aistack (TUI)
# - /usr/local/bin/aistack agent (background service)
```

**Solution**: Only run one at a time:

```bash
# Option A: Use TUI only (stop agent)
sudo systemctl stop aistack-agent

# Then run TUI
aistack

# Option B: Use agent only (recommended for production)
sudo systemctl start aistack-agent

# View status with CLI commands
aistack status
```

#### 2. Background Processes Causing CPU Spikes

The idle detection uses a sliding window average. Even brief CPU spikes can reset the idle timer.

**Check what's using CPU**:
```bash
# Monitor CPU usage in real-time
htop

# Or use top
top
```

Common culprits:
- Docker container pulls/updates
- System updates running in background
- cron jobs
- Other AI services consuming CPU

**Solution**: Wait for background processes to finish, or temporarily increase CPU threshold:

```bash
# Edit config to increase CPU idle threshold
sudo nano /etc/aistack/config.yaml

# Change:
idle:
  cpu_threshold_pct: 10  # Increase to 20 or 30 if you have background noise

# Restart agent
sudo systemctl restart aistack-agent
```

#### 3. tmux/Screen Causing Display Updates

Terminal multiplexers can cause CPU activity from display updates.

**Solution**: Run agent in background instead of TUI:
```bash
# Stop TUI (Ctrl+C or 'q')

# Check agent status via systemd
sudo systemctl status aistack-agent

# View idle state from CLI
aistack status
```

### Verification

After applying fixes, monitor the idle state:

```bash
# Watch idle state in real-time (TUI)
aistack

# Or check via systemd logs
sudo journalctl -u aistack-agent -f | grep idle.state
```

You should see `idle_for_s` increasing over time:
```json
{"type":"idle.state.calculated","idle_for_s":30}
{"type":"idle.state.calculated","idle_for_s":60}
{"type":"idle.state.calculated","idle_for_s":90}
```

---

## GPU Lock Issues

### Symptoms
```
Error: failed to acquire GPU lock: GPU lock is held by <service>
```

### Cause
Multiple services trying to use the GPU simultaneously.

### Solution

This should be fixed in the latest version. Update and reinstall:

```bash
cd ~/aistack
git pull
sudo ./install.sh
sudo aistack remove openwebui --purge
sudo aistack install --profile standard-gpu
```

---

## Common Issues

### Docker Service Not Running

```bash
# Check Docker status
sudo systemctl status docker

# Start Docker if not running
sudo systemctl start docker

# Enable Docker to start at boot
sudo systemctl enable docker
```

### Permission Denied for Docker Commands

```bash
# Add your user to docker group
sudo usermod -aG docker $USER

# Log out and log back in for changes to take effect
```

### CUDA Build Fails

```bash
# Check if NVIDIA driver is installed
nvidia-smi

# If not found, install drivers
sudo ubuntu-drivers autoinstall
sudo reboot

# Install CUDA Toolkit
sudo apt install nvidia-cuda-toolkit

# Rebuild with CUDA support
cd ~/aistack
make clean
make build-cuda
sudo ./install.sh
```

### Health Check Fails After Update

```bash
# Check container logs
aistack logs <service>

# Repair the service
sudo systemctl stop aistack-agent
aistack repair <service>

# Or remove and reinstall
aistack remove <service> --purge
aistack install <service>
```

---

## Debug Mode

Enable debug logging for more detailed troubleshooting:

```bash
# Edit config
sudo nano /etc/aistack/config.yaml

# Add:
logging:
  level: debug

# Restart agent
sudo systemctl restart aistack-agent

# View debug logs
sudo journalctl -u aistack-agent -f
```

---

## Getting Help

If you're still experiencing issues:

1. Create a diagnostic package:
   ```bash
   aistack diag --output /tmp/aistack-diag.zip
   ```

2. Check the GitHub issues: https://github.com/polygonschmiede/aistack/issues

3. Include in your report:
   - Output of `aistack diag`
   - Relevant logs from `journalctl -u aistack-agent`
   - System info: `uname -a` and `nvidia-smi` (if GPU)
