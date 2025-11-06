# Power Management & Wake-on-LAN Guide

> Complete guide to aistack's power efficiency features: idle detection, automatic suspend, and Wake-on-LAN remote wake-up.

## Table of Contents

1. [Overview](#overview)
2. [Idle Detection & Auto-Suspend](#idle-detection--auto-suspend)
3. [Wake-on-LAN Setup](#wake-on-lan-setup)
4. [Configuration](#configuration)
5. [Troubleshooting](#troubleshooting)
6. [Advanced Usage](#advanced-usage)

---

## Overview

aistack provides power-efficient GPU server management through:

- **Idle Detection**: Monitors CPU/GPU utilization with sliding window algorithm
- **Auto-Suspend**: Automatically suspends system to RAM when idle
- **Wake-on-LAN**: Remotely wake system via network magic packet

**Power Savings Example**:
- Idle GPU server: ~200-300W continuous draw
- Suspended to RAM: ~5-10W standby power
- Potential savings: ~$200-300/year (at $0.12/kWh)

**Use Cases**:
- Home lab servers with intermittent use
- Development environments used during work hours only
- Cost-sensitive deployments with variable workloads
- Remote access scenarios with on-demand availability

---

## Idle Detection & Auto-Suspend

### How It Works

aistack uses a sliding window algorithm to determine system idle state:

1. **Metrics Collection**: Every 10 seconds, CPU and GPU utilization are sampled
2. **Window Analysis**: A 5-minute sliding window of samples is maintained
3. **Idle Calculation**: System is idle when both CPU < threshold AND GPU < threshold
4. **Timeout Tracking**: Continuous idle duration is tracked
5. **Suspend Decision**: After idle timeout (default: 30 minutes), system suspends

**Gating Conditions** (prevent premature suspend):
- `warming_up`: Not enough samples collected yet (need 6+ samples)
- `below_timeout`: Idle but timeout not reached yet
- `high_cpu`: CPU utilization above threshold
- `high_gpu`: GPU utilization above threshold
- `inhibit`: systemd inhibitor active (e.g., package updates)

### Check Idle Status

```bash
# View current idle state
cat /var/lib/aistack/idle_state.json | jq .

# Example output:
# {
#   "status": "idle",
#   "idle_for_s": 1850,
#   "threshold_s": 1800,
#   "cpu_idle_pct": 98.5,
#   "gpu_idle_pct": 0.2,
#   "gating_reasons": [],
#   "last_update": "2025-11-06T14:30:00Z"
# }

# Manually trigger idle check
sudo aistack idle-check

# Force check (ignore systemd inhibitors)
sudo aistack idle-check --ignore-inhibitors
```

### Enable Auto-Suspend

**Prerequisites**:
- systemd-based Linux distribution (Ubuntu 24.04 recommended)
- System supports suspend-to-RAM (test with `sudo systemctl suspend`)
- BIOS/UEFI suspend settings enabled

**Setup**:

```bash
# Install aistack agent (enables idle monitoring)
sudo systemctl enable aistack-agent.service
sudo systemctl start aistack-agent.service

# Install idle check timer (triggers suspend decision)
sudo systemctl enable aistack-idle.timer
sudo systemctl start aistack-idle.timer

# Verify agent is running
sudo systemctl status aistack-agent
sudo systemctl status aistack-idle.timer

# Check logs
sudo journalctl -u aistack-agent -f
```

### Configure Thresholds

Edit `/etc/aistack/config.yaml`:

```yaml
idle:
  cpu_idle_threshold: 10      # CPU below 10% = idle
  gpu_idle_threshold: 5       # GPU below 5% = idle
  window_seconds: 300         # 5-minute sliding window
  idle_timeout_seconds: 1800  # Suspend after 30 minutes idle
```

**Tuning Recommendations**:
- **Conservative** (avoid false positives):
  - cpu_idle_threshold: 5
  - gpu_idle_threshold: 2
  - idle_timeout_seconds: 3600 (1 hour)

- **Aggressive** (maximize power savings):
  - cpu_idle_threshold: 15
  - gpu_idle_threshold: 10
  - idle_timeout_seconds: 900 (15 minutes)

- **Balanced** (default):
  - cpu_idle_threshold: 10
  - gpu_idle_threshold: 5
  - idle_timeout_seconds: 1800 (30 minutes)

### Disable Auto-Suspend

```bash
# Stop and disable timer
sudo systemctl stop aistack-idle.timer
sudo systemctl disable aistack-idle.timer

# Agent can keep running for metrics collection
# To also stop agent:
sudo systemctl stop aistack-agent.service
sudo systemctl disable aistack-agent.service
```

---

## Wake-on-LAN Setup

### Prerequisites

**Hardware Requirements**:
- Network card supports Wake-on-LAN (most modern NICs do)
- BIOS/UEFI has WoL enabled (check in BIOS settings)
- System connected via Ethernet (Wi-Fi WoL is unreliable)

**Network Requirements**:
- Network switch supports broadcast packets
- No firewall blocking UDP ports 7 and 9
- Static IP or DHCP reservation recommended

### Step 1: Check WoL Support

```bash
# Check if WoL is supported and enabled
sudo aistack wol-check

# Example output:
# Interface: eno1
# MAC Address: 00:11:22:33:44:55
# WoL Support: Yes
# Current Mode: g (magic packet)
# Status: ENABLED ✓
```

### Step 2: Enable WoL

```bash
# Enable WoL on default interface (auto-detected)
sudo aistack wol-setup

# Or specify interface explicitly
sudo aistack wol-setup eno1

# Verify
sudo aistack wol-check
```

**What this does**:
- Runs `ethtool -s <interface> wol g` (enable magic packet mode)
- Saves configuration to `/var/lib/aistack/wol_config.json`
- Sets up udev rule for persistence across reboots

### Step 3: Make WoL Persistent

The WoL setup creates a udev rule, but you need to ensure it's loaded:

```bash
# Reload udev rules
sudo udevadm control --reload-rules
sudo udevadm trigger

# Verify persistence after reboot
sudo reboot
# After reboot:
sudo aistack wol-check  # Should still show enabled
```

### Step 4: Test Wake-Up

**From another machine on the same network**:

```bash
# Install wake-on-lan tool (if needed)
sudo apt install wakeonlan

# Wake using aistack from another machine with aistack installed
aistack wol-send 00:11:22:33:44:55

# Or use wakeonlan directly
wakeonlan 00:11:22:33:44:55

# Or use aistack with custom broadcast IP
aistack wol-send 00:11:22:33:44:55 192.168.1.255
```

**Test Procedure**:
1. Note your system's MAC address: `ip link show | grep ether`
2. Suspend your system: `sudo systemctl suspend`
3. From another machine: `aistack wol-send <MAC>`
4. System should wake up in 2-5 seconds

### Step 5: Configure in aistack

Edit `/etc/aistack/config.yaml`:

```yaml
wol:
  interface: eno1                      # Your network interface
  mac: "00:11:22:33:44:55"            # Your MAC address
  relay_url: ""                        # Optional: HTTP relay endpoint
```

After editing, apply configuration:

```bash
# Reapply WoL settings
sudo aistack wol-apply

# Or specify interface
sudo aistack wol-apply eno1
```

---

## Configuration

### Complete Configuration Example

`/etc/aistack/config.yaml`:

```yaml
# Container Runtime
container_runtime: docker

# Profile (minimal, standard-gpu, dev)
profile: standard-gpu

# GPU Lock (exclusive GPU access)
gpu_lock: true

# Idle Detection & Auto-Suspend
idle:
  cpu_idle_threshold: 10      # Percent (0-100)
  gpu_idle_threshold: 5       # Percent (0-100)
  window_seconds: 300         # Sliding window size
  idle_timeout_seconds: 1800  # Time before suspend

# Power Estimation
power_estimation:
  baseline_watts: 50          # System idle power draw

# Wake-on-LAN
wol:
  interface: eno1
  mac: "00:11:22:33:44:55"
  relay_url: ""               # Optional HTTP relay

# Logging
logging:
  level: info                 # debug, info, warn, error
  format: json                # json or text

# Model Cache
models:
  keep_cache_on_uninstall: true

# Updates
updates:
  mode: rolling               # rolling or pinned
```

### Environment Variables

Override configuration with environment variables:

```bash
# State directory (for idle state, plans, etc.)
export AISTACK_STATE_DIR=/var/lib/aistack

# Log directory
export AISTACK_LOG_DIR=/var/log/aistack

# Compose files location
export AISTACK_COMPOSE_DIR=/usr/share/aistack/compose

# Version lock file
export AISTACK_VERSIONS_LOCK=/etc/aistack/versions.lock
```

---

## Troubleshooting

### Problem: System Won't Suspend

**Diagnosis**:
```bash
# Check systemd inhibitors
systemd-inhibit --list

# Check idle state
cat /var/lib/aistack/idle_state.json | jq .

# Check agent logs
sudo journalctl -u aistack-agent -n 50

# Test manual suspend
sudo systemctl suspend
```

**Common Causes**:
- **Desktop Environment Running**: GNOME/KDE may inhibit suspend
  - Solution: Use `aistack idle-check --ignore-inhibitors` or run headless
- **Package Manager Active**: APT/DNF operations block suspend
  - Solution: Wait for updates to complete
- **Thresholds Too Strict**: CPU/GPU never below threshold
  - Solution: Increase thresholds in config
- **Not Enough Samples**: System in `warming_up` state
  - Solution: Wait 60+ seconds for samples to accumulate

### Problem: Wake-on-LAN Not Working

**Diagnosis**:
```bash
# Verify WoL enabled
sudo aistack wol-check

# Verify with ethtool
sudo ethtool eno1 | grep -i wake

# Check BIOS settings
# Reboot and enter BIOS/UEFI
# Look for: Wake-on-LAN, Network Boot, PME (Power Management Event)
```

**Common Causes**:
- **BIOS WoL Disabled**: Enable in BIOS under Power Management
- **Network Switch Doesn't Forward Broadcast**: Check switch capabilities
- **Firewall Blocking**: Ensure UDP ports 7 and 9 are open
- **Wi-Fi Connection**: WoL requires Ethernet connection
- **Wrong MAC Address**: Verify with `ip link show`

**Step-by-Step Debug**:
1. Verify ethtool shows "Supports Wake-on: g"
2. Verify ethtool shows "Wake-on: g" (not "d")
3. Test from same subnet (broadcast must reach)
4. Check BIOS settings after cold boot
5. Try different magic packet tool (aistack, wakeonlan, etherwake)

### Problem: System Suspends Too Quickly

**Symptoms**: System suspends during active use

**Diagnosis**:
```bash
# Check current thresholds
grep -A5 "^idle:" /etc/aistack/config.yaml

# Monitor metrics in real-time
tail -f /var/log/aistack/metrics.log | jq .
```

**Solution**:
```bash
# Increase thresholds or timeout
sudo nano /etc/aistack/config.yaml

# Example: More conservative settings
idle:
  cpu_idle_threshold: 5       # Lower threshold = harder to be idle
  gpu_idle_threshold: 2
  idle_timeout_seconds: 3600  # 1 hour instead of 30 min

# Restart agent
sudo systemctl restart aistack-agent
```

### Problem: WoL Works Once, Then Stops

**Cause**: Network card powered off completely after suspend

**Diagnosis**:
```bash
# Check if udev rule is loaded
sudo udevadm info /sys/class/net/eno1 | grep -i wol

# Check if WoL persists after wake
# Wake system via WoL
# Then check:
sudo aistack wol-check
```

**Solution**:
```bash
# Ensure udev rule is correct
cat /etc/udev/rules.d/50-aistack-wol.rules

# Should contain:
# ACTION=="add", SUBSYSTEM=="net", NAME=="eno1", RUN+="/usr/bin/aistack wol-apply eno1"

# Reload and trigger
sudo udevadm control --reload-rules
sudo udevadm trigger

# Test full suspend/wake/suspend cycle
```

---

## Advanced Usage

### Manual Idle State Management

```bash
# View raw idle state
cat /var/lib/aistack/idle_state.json

# Reset idle state (force restart of idle tracking)
sudo systemctl stop aistack-agent
sudo rm /var/lib/aistack/idle_state.json
sudo systemctl start aistack-agent
```

### Custom Suspend Scripts

Create systemd unit drop-in for pre-suspend actions:

```bash
# Create override directory
sudo mkdir -p /etc/systemd/system/systemd-suspend.service.d

# Create pre-suspend script
sudo nano /etc/systemd/system/systemd-suspend.service.d/aistack-pre-suspend.conf
```

Content:
```ini
[Service]
ExecStartPre=/usr/local/bin/aistack-pre-suspend.sh
```

Create `/usr/local/bin/aistack-pre-suspend.sh`:
```bash
#!/bin/bash
# Custom actions before suspend
logger "aistack: Preparing for suspend"
# e.g., flush logs, close connections, etc.
```

Make executable:
```bash
sudo chmod +x /usr/local/bin/aistack-pre-suspend.sh
sudo systemctl daemon-reload
```

### Wake-on-LAN HTTP Relay

For remote wake-up from outside your local network:

```bash
# Start WoL relay server on a machine in your local network
aistack wol-relay --port 8090 --key your-secret-key

# Or use environment variable
export AISTACK_WOL_RELAY_KEY=your-secret-key
aistack wol-relay --port 8090
```

From anywhere (with key):
```bash
curl -X POST http://your-server:8090/wake \
  -H "X-API-Key: your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"mac": "00:11:22:33:44:55"}'
```

**Security Note**: Always use a strong API key and consider HTTPS with reverse proxy.

### Metrics Monitoring

```bash
# View live metrics
tail -f /var/log/aistack/metrics.log | jq .

# Extract CPU utilization over last hour
grep -o '"cpu_util":[0-9.]*' /var/log/aistack/metrics.log | \
  awk -F: '{print $2}' | \
  awk '{sum+=$1; count++} END {print sum/count}'

# Extract GPU power draw
grep -o '"gpu_w":[0-9.]*' /var/log/aistack/metrics.log | \
  awk -F: '{print $2}'
```

---

## Best Practices

1. **Test Before Production**:
   - Test suspend/wake cycle manually before enabling auto-suspend
   - Verify WoL works from expected client machines

2. **Conservative Initial Settings**:
   - Start with 1-hour idle timeout
   - Monitor for false suspends over a week
   - Gradually reduce timeout as confidence increases

3. **Network Setup**:
   - Use static IP or DHCP reservation
   - Document MAC addresses in inventory
   - Test WoL from multiple subnets if needed

4. **Monitoring**:
   - Set up alerts for unexpected suspends
   - Monitor metrics to understand usage patterns
   - Review idle state logs regularly

5. **Documentation**:
   - Document your specific hardware quirks
   - Keep record of BIOS settings
   - Note any custom systemd overrides

6. **Backup Plan**:
   - Keep physical access available
   - Document manual wake procedure (power button)
   - Have out-of-band management (IPMI) if available

---

## FAQ

**Q: Can I use aistack power management with a desktop environment?**
A: Yes, but desktop environments (GNOME, KDE) may conflict with suspend decisions. Use `--ignore-inhibitors` flag or configure your DE to allow system suspend.

**Q: Does suspend affect running AI workloads?**
A: Yes. Suspend pauses all processes. Workloads resume after wake-up. For long-running tasks, disable auto-suspend or increase timeout.

**Q: Can I wake the system from the internet?**
A: Not directly (WoL is broadcast-only). Use the WoL HTTP relay on a machine in your local network, or set up port forwarding with Wake-on-WAN capable routers.

**Q: What happens if wake fails?**
A: Physical power button press is always the fallback. Some systems also support IPMI/BMC for out-of-band management.

**Q: How much power does suspend-to-RAM use?**
A: Typically 5-10W for the entire system (RAM stays powered, everything else off).

**Q: Will suspend wear out my RAM?**
A: No. Suspend-to-RAM (S3 sleep) is designed for frequent use and does not degrade RAM.

**Q: Can I use hibernation instead of suspend?**
A: aistack currently uses suspend-to-RAM (faster wake, but requires power). Hibernation support may be added in future versions.

---

For operational procedures and troubleshooting, see [OPERATIONS.md](OPERATIONS.md).

For general usage and installation, see [README.md](../README.md).
