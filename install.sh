#!/usr/bin/env bash
#
# aistack Bootstrap Installer
# Headless installation script for Ubuntu 24.04
# Checks system requirements, installs Docker, deploys systemd units
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

CONTAINER_RUNTIME=""

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_event() {
    local event_type="$1"
    local payload="$2"
    echo "{\"type\":\"${event_type}\",\"payload\":${payload},\"ts\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" >> /tmp/aistack-bootstrap.log
}

# Check if running with sudo/root
check_sudo() {
    log_info "Checking for root/sudo privileges..."

    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run with sudo privileges"
        log_error "Please run: sudo bash $0"
        exit 1
    fi

    log_info "✓ Running with required privileges"
}

# Check OS version (Ubuntu 24.04 required)
check_os_version() {
    log_info "Checking OS version..."

    if [[ ! -f /etc/os-release ]]; then
        log_error "/etc/os-release not found. Unable to determine OS."
        exit 1
    fi

    source /etc/os-release

    if [[ "$ID" != "ubuntu" ]]; then
        log_error "This script requires Ubuntu. Detected: $ID"
        log_error "Currently only Ubuntu 24.04 is supported."
        exit 1
    fi

    if [[ "$VERSION_ID" != "24.04" ]]; then
        log_error "This script requires Ubuntu 24.04. Detected: $VERSION_ID"
        log_error "Please upgrade to Ubuntu 24.04 before proceeding."
        exit 1
    fi

    log_info "✓ OS check passed: Ubuntu $VERSION_ID"
    log_event "bootstrap.checks" "{\"os\":\"ubuntu-$VERSION_ID\",\"sudo\":true}"
}

# Check internet connectivity
check_internet() {
    log_info "Checking internet connectivity..."

    if ! ping -c 1 -W 2 8.8.8.8 &> /dev/null; then
        log_error "No internet connectivity detected"
        log_error "Please ensure you have a working internet connection and try again."
        exit 1
    fi

    log_info "✓ Internet connectivity verified"
}

# Ensure Docker or Podman is available before proceeding
ensure_container_runtime() {
    log_info "Checking container runtime..."

    if command -v docker &> /dev/null; then
        CONTAINER_RUNTIME="docker"
        log_info "Docker detected"

        # Always enable and start (idempotent in systemctl)
        log_info "Ensuring Docker service is enabled and running..."
        systemctl enable docker
        systemctl start docker
        sleep 2

        if systemctl is-active --quiet docker; then
            log_info "✓ Docker service is running"
        else
            log_error "Failed to start Docker service"
            exit 1
        fi

        log_event "bootstrap.runtime" "{\"runtime\":\"docker\",\"state\":\"running\"}"
        return 0
    fi

    if command -v podman &> /dev/null; then
        CONTAINER_RUNTIME="podman"
        log_info "Podman detected — using Podman (best-effort support)"
        log_event "bootstrap.runtime" "{\"runtime\":\"podman\",\"state\":\"detected\"}"
        return 0
    fi

    log_info "No container runtime found, installing Docker..."
    install_docker
}

# Install Docker
install_docker() {
    log_info "Installing Docker (always fresh install)..."

    # Update package index
    log_info "Updating package index..."
    apt-get update -qq

    # Install prerequisites
    log_info "Installing prerequisites..."
    apt-get install -y -qq \
        ca-certificates \
        curl \
        gnupg \
        lsb-release \
        &> /dev/null

    # Add Docker's official GPG key
    log_info "Adding Docker GPG key..."
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg

    # Set up Docker repository
    log_info "Adding Docker repository..."
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

    # Install Docker Engine
    log_info "Installing Docker Engine..."
    apt-get update -qq
    apt-get install -y -qq \
        docker-ce \
        docker-ce-cli \
        containerd.io \
        docker-buildx-plugin \
        docker-compose-plugin \
        &> /dev/null

    # Enable and start Docker service
    log_info "Enabling and starting Docker service..."
    systemctl enable --now docker

    # Wait for Docker to be ready
    sleep 2

    # Verify installation
    if systemctl is-active --quiet docker; then
        local docker_version=$(docker --version | awk '{print $3}' | tr -d ',')
        log_info "✓ Docker installed successfully (version: $docker_version)"
        log_event "bootstrap.docker.installed" "{\"version\":\"$docker_version\"}"

        CONTAINER_RUNTIME="docker"
        log_event "bootstrap.runtime" "{\"runtime\":\"docker\",\"state\":\"installed\"}"

        # Test Docker with hello-world
        log_info "Running Docker test container..."
        if docker run --rm hello-world &> /dev/null; then
            log_info "✓ Docker test successful"
        else
            log_warn "Docker test container failed, but service is running"
        fi

        return 0
    else
        log_error "Docker installation failed - service is not active"
        exit 1
    fi
}

# Build aistack binary (auto-detects CUDA) - ALWAYS rebuild
build_aistack_binary() {
    log_info "Building aistack binary (always fresh build)..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Always clean and rebuild
    log_info "Cleaning old build artifacts..."
    (cd "$script_dir" && rm -rf ./dist)

    # Build with auto-detection
    if command -v make &> /dev/null; then
        log_info "Building with auto-detection (make will detect CUDA if available)..."
        (cd "$script_dir" && make build) || {
            log_error "Build failed"
            exit 1
        }
    else
        log_warn "make not found, falling back to basic build without GPU support"
        mkdir -p "$script_dir/dist"
        (cd "$script_dir" && CGO_ENABLED=0 go build -tags netgo -ldflags "-s -w" -o ./dist/aistack ./cmd/aistack)
    fi

    log_info "✓ Binary built successfully"
}

# Install or update the aistack CLI binary under /usr/local/bin - ALWAYS reinstall
install_cli_binary() {
    log_info "Installing aistack CLI binary (always fresh install)..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Always rebuild
    build_aistack_binary

    local candidates=(
        "$script_dir/dist/aistack"
        "$script_dir/../dist/aistack"
        "$script_dir/aistack"
    )

    local source=""
    for candidate in "${candidates[@]}"; do
        if [[ -f "$candidate" ]]; then
            source="$candidate"
            break
        fi
    done

    if [[ -z "$source" ]]; then
        log_error "aistack binary not found after build. Please check build errors."
        exit 1
    fi

    # Always overwrite
    log_info "Copying binary to /usr/local/bin/aistack..."
    install -m 0755 "$source" /usr/local/bin/aistack
    log_info "✓ Installed CLI to /usr/local/bin/aistack"
}

# Create/overwrite /etc/aistack/config.yaml with defaults - ALWAYS overwrite
ensure_config_defaults() {
    local config_path="/etc/aistack/config.yaml"

    log_info "Writing fresh config to $config_path (always overwrite)..."

    cat > "$config_path" <<CONFIG
container_runtime: ${CONTAINER_RUNTIME:-docker}
profile: standard-gpu
gpu_lock: true

# Power Management & Idle Detection
idle:
  window_seconds: 60
  idle_timeout_seconds: 300
  cpu_threshold_pct: 10.0
  gpu_threshold_pct: 5.0
  min_samples_required: 6
  enable_suspend: true

# Update Policy
updates:
  mode: rolling

# Logging
logging:
  level: info
CONFIG

    chown root:aistack "$config_path"
    chmod 640 "$config_path"
    log_info "✓ Created default config at $config_path"
}

# Deploy udev rules for persistent Wake-on-LAN configuration and RAPL permissions - ALWAYS redeploy
deploy_udev_rules() {
    log_info "Deploying udev rules (always overwrite)..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Deploy WoL rule
    local wol_source="$script_dir/assets/udev/70-aistack-wol.rules"
    local wol_target="/etc/udev/rules.d/70-aistack-wol.rules"

    if [[ -f "$wol_source" ]]; then
        cp -f "$wol_source" "$wol_target"
        chmod 644 "$wol_target"
        log_info "✓ Deployed udev rule: $(basename "$wol_target")"
    else
        log_warn "WoL udev rule not found: $wol_source"
    fi

    # Deploy RAPL rule (for CPU power monitoring)
    local rapl_source="$script_dir/assets/udev/80-aistack-rapl.rules"
    local rapl_target="/etc/udev/rules.d/80-aistack-rapl.rules"

    if [[ -f "$rapl_source" ]]; then
        cp -f "$rapl_source" "$rapl_target"
        chmod 644 "$rapl_target"
        log_info "✓ Deployed udev rule: $(basename "$rapl_target")"
    else
        log_warn "RAPL udev rule not found: $rapl_source"
    fi

    # Always reload udev rules and trigger
    if command -v udevadm &> /dev/null; then
        log_info "Reloading and triggering udev rules..."
        udevadm control --reload-rules
        udevadm trigger --subsystem-match=net || true
        udevadm trigger --subsystem-match=powercap || true
        log_info "✓ Reloaded and triggered udev rules"
    fi
}

# Deploy systemd-tmpfiles configuration for RAPL permissions - ALWAYS redeploy
deploy_tmpfiles() {
    log_info "Deploying systemd-tmpfiles configuration (always overwrite)..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local tmpfiles_source="$script_dir/assets/tmpfiles.d/aistack-rapl.conf"
    local tmpfiles_target="/etc/tmpfiles.d/aistack-rapl.conf"

    if [[ ! -f "$tmpfiles_source" ]]; then
        log_warn "tmpfiles config not found: $tmpfiles_source"
        return
    fi

    cp -f "$tmpfiles_source" "$tmpfiles_target"
    chmod 644 "$tmpfiles_target"
    log_info "✓ Deployed tmpfiles config: $(basename "$tmpfiles_target")"

    # Always apply tmpfiles configuration immediately
    if command -v systemd-tmpfiles &> /dev/null; then
        log_info "Applying tmpfiles configuration..."
        systemd-tmpfiles --create "$tmpfiles_target" || {
            log_warn "Failed to apply tmpfiles config (non-critical)"
        }
        log_info "✓ Applied tmpfiles configuration"
    fi
}

# Create aistack user and group - ALWAYS recreate
create_aistack_user() {
    log_info "Ensuring aistack user and group exist..."

    # Create user if not exists, update if exists
    if id "aistack" &>/dev/null; then
        log_info "User 'aistack' exists, ensuring configuration is correct..."
    else
        useradd -r -s /bin/false -d /var/lib/aistack aistack
        log_info "✓ Created system user 'aistack'"
    fi

    # Always ensure docker group membership
    log_info "Ensuring 'aistack' user is in docker group..."
    usermod -aG docker aistack 2>/dev/null || true
    log_info "✓ User 'aistack' configured for docker access"
}

# Create necessary directories - ALWAYS recreate and set permissions
create_directories() {
    log_info "Creating directory structure (always fresh)..."

    local dirs=(
        "/var/lib/aistack"
        "/var/log/aistack"
        "/etc/aistack"
    )

    for dir in "${dirs[@]}"; do
        log_info "Ensuring directory exists: $dir"
        mkdir -p "$dir"
    done

    # Always set ownership and permissions
    log_info "Setting directory ownership and permissions..."
    chown -R aistack:aistack /var/lib/aistack
    chown -R aistack:aistack /var/log/aistack
    chown -R root:aistack /etc/aistack
    chmod 750 /etc/aistack

    log_info "✓ Directory structure configured"
}

# Deploy Wake-on-LAN persistence
deploy_wol_persistence() {
    log_info "Deploying Wake-on-LAN persistence service..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local scripts_dir="${script_dir}/assets/scripts"
    local systemd_dir="${script_dir}/assets/systemd"

    # Check if directories exist
    if [[ ! -d "$scripts_dir" ]]; then
        log_error "scripts directory not found: $scripts_dir"
        exit 1
    fi
    if [[ ! -d "$systemd_dir" ]]; then
        log_error "systemd directory not found: $systemd_dir"
        exit 1
    fi

    # Deploy WoL enable script
    local wol_script="${scripts_dir}/enable-wol.sh"
    if [[ ! -f "$wol_script" ]]; then
        log_error "WoL script not found: $wol_script"
        exit 1
    fi

    cp -f "$wol_script" /usr/local/bin/aistack-enable-wol.sh
    chmod 755 /usr/local/bin/aistack-enable-wol.sh

    # Deploy systemd service
    cp -f "${systemd_dir}/aistack-wol-persist.service" /etc/systemd/system/
    chmod 644 /etc/systemd/system/aistack-wol-persist.service

    # Reload systemd daemon
    systemctl daemon-reload

    # Enable and start service
    systemctl enable aistack-wol-persist.service
    systemctl start aistack-wol-persist.service

    log_info "✓ Wake-on-LAN persistence deployed and started"
    log_info "  WoL will be automatically enabled on boot and after suspend"
    log_info "  Check status: systemctl status aistack-wol-persist.service"
}

# Deploy systemd units for auto-suspend
deploy_systemd_units() {
    log_info "Deploying systemd units for auto-suspend..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local systemd_dir="${script_dir}/assets/systemd"

    # Check if systemd directory exists
    if [[ ! -d "$systemd_dir" ]]; then
        log_error "systemd directory not found: $systemd_dir"
        exit 1
    fi

    # Deploy service and timer
    cp -f "${systemd_dir}/aistack-suspend.service" /etc/systemd/system/
    cp -f "${systemd_dir}/aistack-suspend.timer" /etc/systemd/system/
    chmod 644 /etc/systemd/system/aistack-suspend.service
    chmod 644 /etc/systemd/system/aistack-suspend.timer

    # Reload systemd daemon
    systemctl daemon-reload

    # Enable and start timer (not service - timer triggers service)
    systemctl enable aistack-suspend.timer
    systemctl start aistack-suspend.timer

    log_info "✓ systemd units deployed and timer started"
    log_info "  Auto-suspend will activate after 5 minutes of idle time"
    log_info "  Check status: systemctl status aistack-suspend.timer"
    log_info "  Disable: aistack suspend disable"
}

# Deploy logrotate configuration - ALWAYS redeploy
deploy_logrotate() {
    log_info "Deploying logrotate configuration (always overwrite)..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local logrotate_source="${script_dir}/assets/logrotate/aistack"

    if [[ ! -f "$logrotate_source" ]]; then
        log_error "logrotate config not found: $logrotate_source"
        exit 1
    fi

    cp -f "$logrotate_source" /etc/logrotate.d/aistack
    chmod 644 /etc/logrotate.d/aistack
    log_info "✓ Deployed logrotate configuration"

    # Test logrotate configuration
    if logrotate -d /etc/logrotate.d/aistack &> /dev/null; then
        log_info "✓ Logrotate configuration is valid"
    else
        log_warn "Logrotate configuration test failed (non-critical)"
    fi
}

# Main installation flow
main() {
    echo "========================================"
    echo "  aistack bootstrap Installer"
    echo "========================================"
    echo ""

    # Run all checks
    check_sudo
    check_os_version
    check_internet

    # Ensure container runtime
    ensure_container_runtime

    # Install CLI binary for system usage
    install_cli_binary

    # Setup user and directories
    create_aistack_user
    create_directories
    ensure_config_defaults

    # Deploy configurations
    deploy_logrotate
    deploy_udev_rules
    deploy_tmpfiles
    deploy_wol_persistence
    deploy_systemd_units

    echo ""
    log_info "========================================="
    log_info "Bootstrap completed successfully!"
    log_info "========================================="
    log_info ""
    log_info "Next steps:"
    log_info "  1. Run aistack CLI: aistack --help"
    log_info "  2. Start TUI: aistack"
    log_info "  3. Check service status: aistack status"
    log_info ""
    log_info "Bootstrap log: /tmp/aistack-bootstrap.log"
}

# Run main function
main "$@"
