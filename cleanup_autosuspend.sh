#!/usr/bin/env bash
#
# aistack Auto-Suspend Feature Cleanup Script
#
# This script removes all components of the deprecated auto-suspend feature:
# - systemd services (agent, idle timer)
# - State files (idle state, WoL config)
# - Old config sections (requires manual verification)
#
# Run with sudo: sudo bash cleanup_autosuspend.sh
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Check if running with sudo/root
check_sudo() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run with sudo privileges"
        log_error "Please run: sudo bash $0"
        exit 1
    fi
    log_info "Running with root privileges ✓"
}

# Stop and disable systemd services
stop_systemd_services() {
    log_info "Stopping and disabling old systemd services..."

    local services=(
        "aistack-agent.service"
        "aistack-idle.service"
        "aistack-idle.timer"
    )

    for service in "${services[@]}"; do
        if systemctl is-active --quiet "$service" 2>/dev/null; then
            log_info "  Stopping $service..."
            systemctl stop "$service" || log_warn "Failed to stop $service"
        fi

        if systemctl is-enabled --quiet "$service" 2>/dev/null; then
            log_info "  Disabling $service..."
            systemctl disable "$service" || log_warn "Failed to disable $service"
        fi
    done

    log_success "Systemd services stopped and disabled"
}

# Remove systemd unit files
remove_systemd_units() {
    log_info "Removing systemd unit files..."

    local units=(
        "/etc/systemd/system/aistack-agent.service"
        "/etc/systemd/system/aistack-idle.service"
        "/etc/systemd/system/aistack-idle.timer"
    )

    for unit in "${units[@]}"; do
        if [[ -f "$unit" ]]; then
            log_info "  Removing $unit..."
            rm -f "$unit"
        fi
    done

    log_success "Systemd unit files removed"
}

# Reload systemd daemon
reload_systemd() {
    log_info "Reloading systemd daemon..."
    systemctl daemon-reload
    log_success "Systemd daemon reloaded"
}

# Remove state files
remove_state_files() {
    log_info "Removing state files..."

    # Determine state directory
    local state_dir="${AISTACK_STATE_DIR:-/var/lib/aistack}"

    if [[ ! -d "$state_dir" ]]; then
        log_warn "State directory not found: $state_dir"
        return 0
    fi

    local state_files=(
        "$state_dir/idle_state.json"
        "$state_dir/wol_config.json"
    )

    for file in "${state_files[@]}"; do
        if [[ -f "$file" ]]; then
            log_info "  Removing $file..."
            rm -f "$file"
        fi
    done

    log_success "State files removed"
}

# Check and warn about config files
check_config_files() {
    log_info "Checking configuration files..."

    local config_files=(
        "/etc/aistack/config.yaml"
        "$HOME/.aistack/config.yaml"
    )

    local found_config=false

    for config in "${config_files[@]}"; do
        if [[ -f "$config" ]]; then
            log_warn "Found config file: $config"

            # Check if it contains deprecated sections
            if grep -q "^idle:" "$config" 2>/dev/null || \
               grep -q "^power_estimation:" "$config" 2>/dev/null || \
               grep -q "^wol:" "$config" 2>/dev/null; then
                log_warn "  ⚠ This config contains deprecated sections (idle/power_estimation/wol)"
                log_warn "  → Please remove these sections manually"
                found_config=true
            fi
        fi
    done

    if $found_config; then
        echo ""
        log_warn "⚠ Manual action required:"
        log_warn "Edit your config files and remove the following sections:"
        log_warn "  - idle:"
        log_warn "  - power_estimation:"
        log_warn "  - wol:"
        echo ""
    else
        log_success "No deprecated config sections found"
    fi
}

# Verify cleanup
verify_cleanup() {
    log_info "Verifying cleanup..."

    local issues=0

    # Check systemd services
    local services=("aistack-agent.service" "aistack-idle.service" "aistack-idle.timer")
    for service in "${services[@]}"; do
        if systemctl is-active --quiet "$service" 2>/dev/null; then
            log_error "  ✗ Service still active: $service"
            ((issues++))
        fi
    done

    # Check unit files
    local units=(
        "/etc/systemd/system/aistack-agent.service"
        "/etc/systemd/system/aistack-idle.service"
        "/etc/systemd/system/aistack-idle.timer"
    )
    for unit in "${units[@]}"; do
        if [[ -f "$unit" ]]; then
            log_error "  ✗ Unit file still exists: $unit"
            ((issues++))
        fi
    done

    # Check state files
    local state_dir="${AISTACK_STATE_DIR:-/var/lib/aistack}"
    local state_files=("$state_dir/idle_state.json" "$state_dir/wol_config.json")
    for file in "${state_files[@]}"; do
        if [[ -f "$file" ]]; then
            log_error "  ✗ State file still exists: $file"
            ((issues++))
        fi
    done

    if [[ $issues -eq 0 ]]; then
        log_success "Cleanup verification passed ✓"
        return 0
    else
        log_error "Cleanup verification failed: $issues issues found"
        return 1
    fi
}

# Main execution
main() {
    echo "=========================================="
    echo "  aistack Auto-Suspend Feature Cleanup"
    echo "=========================================="
    echo ""

    log_info "This script will remove all auto-suspend/idle/WoL components"
    echo ""

    # Confirm before proceeding
    if [[ "${1:-}" != "--yes" ]]; then
        read -rp "Do you want to proceed? (yes/no): " response
        if [[ "$response" != "yes" ]]; then
            log_warn "Cleanup canceled by user"
            exit 0
        fi
        echo ""
    fi

    # Run cleanup steps
    check_sudo
    stop_systemd_services
    remove_systemd_units
    reload_systemd
    remove_state_files

    echo ""
    check_config_files

    echo ""
    verify_cleanup

    echo ""
    log_success "=========================================="
    log_success "Cleanup completed successfully!"
    log_success "=========================================="
    echo ""
    log_info "Next steps:"
    log_info "  1. Verify config files and remove deprecated sections"
    log_info "  2. Restart any aistack services if needed"
    log_info "  3. Update aistack binary to latest version"
    echo ""
}

# Run main function
main "$@"
