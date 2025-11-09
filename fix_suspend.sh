#!/usr/bin/env bash
#
# Quick Fix für Suspend Problem
# Behebt fehlende idle: config und ignoriert system inhibitors
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Check if running with sudo/root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run with sudo privileges"
    log_error "Please run: sudo bash $0"
    exit 1
fi

echo "========================================="
echo "  aistack Suspend Quick Fix"
echo "========================================="
echo ""

# 1. Fix Config - Add idle section if missing
log_info "Step 1: Fixing /etc/aistack/config.yaml..."

if grep -q "^idle:" /etc/aistack/config.yaml 2>/dev/null; then
    log_info "✓ Config already has idle: section"
else
    log_warn "Config missing idle: section, adding it..."

    cat >> /etc/aistack/config.yaml << 'EOF'

# Power Management & Idle Detection
idle:
  window_seconds: 60
  idle_timeout_seconds: 300
  cpu_threshold_pct: 10.0
  gpu_threshold_pct: 5.0
  min_samples_required: 6
  enable_suspend: true

# Logging
logging:
  level: info
EOF

    log_info "✓ Added idle: section to config"
fi

# 2. Fix systemd service - Add --ignore-inhibitors
log_info "Step 2: Updating aistack-idle.service..."

if grep -q "\-\-ignore-inhibitors" /etc/systemd/system/aistack-idle.service 2>/dev/null; then
    log_info "✓ Service already uses --ignore-inhibitors"
else
    log_warn "Service missing --ignore-inhibitors, adding it..."

    sed -i 's|ExecStart=/usr/local/bin/aistack idle-check|ExecStart=/usr/local/bin/aistack idle-check --ignore-inhibitors|' /etc/systemd/system/aistack-idle.service

    log_info "✓ Updated service to use --ignore-inhibitors"
fi

# 3. Fix RAPL permissions (if tmpfiles exists)
log_info "Step 3: Fixing RAPL permissions..."

if [[ -f "/etc/tmpfiles.d/aistack-rapl.conf" ]]; then
    log_info "✓ RAPL tmpfiles config already exists"
    systemd-tmpfiles --create /etc/tmpfiles.d/aistack-rapl.conf || log_warn "Failed to apply tmpfiles (non-critical)"
else
    log_warn "RAPL tmpfiles config not found (needs newer version)"
    log_info "You can fix this with: git pull && sudo ./install.sh"
fi

# 4. Reload systemd
log_info "Step 4: Reloading systemd..."
systemctl daemon-reload
log_info "✓ Systemd reloaded"

# 5. Restart services
log_info "Step 5: Restarting services..."
systemctl restart aistack-agent
log_info "✓ Restarted aistack-agent"

# 6. Verify
log_info "Step 6: Verifying configuration..."

echo ""
echo "=== Current Configuration ==="
echo ""

echo "Config idle section:"
grep -A 6 "^idle:" /etc/aistack/config.yaml || echo "NOT FOUND!"
echo ""

echo "Service idle-check command:"
grep "ExecStart" /etc/systemd/system/aistack-idle.service || echo "NOT FOUND!"
echo ""

echo "Timer status:"
systemctl status aistack-idle.timer --no-pager | head -5
echo ""

echo "Agent status:"
systemctl status aistack-agent --no-pager | head -5
echo ""

# 7. Show current idle state
log_info "Current idle state:"
cat /var/lib/aistack/idle_state.json 2>/dev/null | python3 -m json.tool || log_warn "No idle state yet"
echo ""

# 8. Final instructions
echo ""
log_info "========================================="
log_info "Fix completed successfully!"
log_info "========================================="
echo ""
log_info "Next steps:"
log_info "  1. Close this SSH session (exit)"
log_info "  2. Wait 5 minutes"
log_info "  3. Server should suspend automatically"
log_info "  4. Test wake-up with WoL magic packet"
log_info ""
log_info "To test immediately (will kick you out of SSH):"
log_info "  sudo aistack idle-check --ignore-inhibitors"
log_info ""
log_info "Monitor logs:"
log_info "  sudo journalctl -u aistack-idle.service -f"
log_info ""
