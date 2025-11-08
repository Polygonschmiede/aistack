#!/usr/bin/env bash
#
# aistack Bootstrap Installer (EP-002)
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
    log_info "Ensuring container runtime availability..."

    if check_docker_installed; then
        CONTAINER_RUNTIME="docker"
        log_event "bootstrap.runtime" "{\"runtime\":\"docker\",\"state\":\"detected\"}"
        return 0
    fi

    if command -v podman &> /dev/null; then
        CONTAINER_RUNTIME="podman"
        log_info "Podman detected — Docker installation skipped (best-effort support)"
        log_event "bootstrap.runtime" "{\"runtime\":\"podman\",\"state\":\"detected\"}"
        return 0
    fi

    install_docker
}

# Check if Docker is already installed and running
check_docker_installed() {
    if command -v docker &> /dev/null; then
        local docker_version=$(docker --version | awk '{print $3}' | tr -d ',')
        log_info "Docker is already installed (version: $docker_version)"

        if systemctl is-active --quiet docker; then
            log_info "✓ Docker service is active (running)"
            return 0
        else
            log_warn "Docker is installed but not running. Starting service..."
            systemctl enable --now docker
            sleep 2

            if systemctl is-active --quiet docker; then
                log_info "✓ Docker service started successfully"
                return 0
            else
                log_error "Failed to start Docker service"
                return 1
            fi
        fi
    fi

    return 1
}

# Install Docker
install_docker() {
    log_info "Installing Docker..."

    # Check if already installed (idempotent)
    if check_docker_installed; then
        log_info "Docker installation check: already installed and running (idempotent)"
        return 0
    fi

    log_info "Docker not found. Proceeding with installation..."

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

# Detect GPU and build with appropriate flags
build_aistack_binary() {
    log_info "Building aistack binary..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Check if binary already exists
    if [[ -f "$script_dir/dist/aistack" ]]; then
        log_info "✓ Binary already exists: $script_dir/dist/aistack"
        return
    fi

    # Detect NVIDIA GPU
    if command -v nvidia-smi &> /dev/null; then
        log_info "NVIDIA GPU detected, attempting CUDA build..."

        # Check for CUDA toolkit
        if [[ -d "/usr/local/cuda" ]] || [[ -d "/usr/lib/cuda" ]] || command -v nvcc &> /dev/null; then
            log_info "CUDA Toolkit found, building with GPU support..."
            if command -v make &> /dev/null; then
                (cd "$script_dir" && make build-cuda) || {
                    log_warn "CUDA build failed, falling back to non-GPU build"
                    (cd "$script_dir" && make build)
                }
            else
                log_warn "make not found, falling back to basic build"
                (cd "$script_dir" && CGO_ENABLED=0 go build -tags netgo -ldflags "-s -w" -o ./dist/aistack ./cmd/aistack)
            fi
        else
            log_info "CUDA Toolkit not found, building without GPU support"
            log_info "For GPU support, install: sudo apt install nvidia-cuda-toolkit"
            (cd "$script_dir" && make build) || {
                (cd "$script_dir" && CGO_ENABLED=0 go build -tags netgo -ldflags "-s -w" -o ./dist/aistack ./cmd/aistack)
            }
        fi
    else
        log_info "No NVIDIA GPU detected, building without GPU support"
        if command -v make &> /dev/null; then
            (cd "$script_dir" && make build)
        else
            (cd "$script_dir" && CGO_ENABLED=0 go build -tags netgo -ldflags "-s -w" -o ./dist/aistack ./cmd/aistack)
        fi
    fi

    log_info "✓ Binary built successfully"
}

# Install or update the aistack CLI binary under /usr/local/bin
install_cli_binary() {
    log_info "Installing aistack CLI binary..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Build if binary doesn't exist
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

    install -m 0755 "$source" /usr/local/bin/aistack
    log_info "✓ Installed CLI to /usr/local/bin/aistack"
}

# Ensure /etc/aistack/config.yaml exists with basic defaults
ensure_config_defaults() {
    local config_path="/etc/aistack/config.yaml"

    if [[ -f "$config_path" ]]; then
        log_info "✓ Existing config preserved: $config_path"
        return
    fi

    cat > "$config_path" <<CONFIG
container_runtime: ${CONTAINER_RUNTIME:-docker}
profile: standard-gpu
gpu_lock: true
updates:
  mode: rolling
CONFIG

    chown root:aistack "$config_path"
    chmod 640 "$config_path"
    log_info "✓ Created default config at $config_path"
}

# Deploy udev rules for persistent Wake-on-LAN configuration and RAPL permissions
deploy_udev_rules() {
    log_info "Deploying udev rules..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Deploy WoL rule
    local wol_source="$script_dir/assets/udev/70-aistack-wol.rules"
    local wol_target="/etc/udev/rules.d/70-aistack-wol.rules"

    if [[ -f "$wol_source" ]]; then
        cp "$wol_source" "$wol_target"
        chmod 644 "$wol_target"
        log_info "✓ Deployed udev rule: $(basename "$wol_target")"
    else
        log_warn "WoL udev rule not found: $wol_source"
    fi

    # Deploy RAPL rule (for CPU power monitoring)
    local rapl_source="$script_dir/assets/udev/80-aistack-rapl.rules"
    local rapl_target="/etc/udev/rules.d/80-aistack-rapl.rules"

    if [[ -f "$rapl_source" ]]; then
        cp "$rapl_source" "$rapl_target"
        chmod 644 "$rapl_target"
        log_info "✓ Deployed udev rule: $(basename "$rapl_target")"
    else
        log_warn "RAPL udev rule not found: $rapl_source"
    fi

    # Reload udev rules
    if command -v udevadm &> /dev/null; then
        udevadm control --reload-rules
        udevadm trigger --subsystem-match=net || true
        udevadm trigger --subsystem-match=powercap || true
        log_info "✓ Reloaded udev rules"
    fi
}

# Create aistack user and group
create_aistack_user() {
    log_info "Creating aistack user and group..."

    if id "aistack" &>/dev/null; then
        log_info "✓ User 'aistack' already exists (idempotent)"
    else
        useradd -r -s /bin/false -d /var/lib/aistack aistack
        log_info "✓ Created system user 'aistack'"
    fi

    # Add aistack user to docker group
    if groups aistack | grep -q docker; then
        log_info "✓ User 'aistack' already in docker group"
    else
        usermod -aG docker aistack
        log_info "✓ Added 'aistack' to docker group"
    fi
}

# Create necessary directories
create_directories() {
    log_info "Creating directory structure..."

    local dirs=(
        "/var/lib/aistack"
        "/var/log/aistack"
        "/etc/aistack"
    )

    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            log_info "✓ Directory exists: $dir"
        else
            mkdir -p "$dir"
            log_info "✓ Created directory: $dir"
        fi
    done

    # Set ownership
    chown -R aistack:aistack /var/lib/aistack
    chown -R aistack:aistack /var/log/aistack
    chown -R root:aistack /etc/aistack
    chmod 750 /etc/aistack

    log_info "✓ Directory permissions configured"
}

# Deploy systemd units
deploy_systemd_units() {
    log_info "Deploying systemd units..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local systemd_source="${script_dir}/assets/systemd"

    if [[ ! -d "$systemd_source" ]]; then
        log_error "systemd units directory not found: $systemd_source"
        exit 1
    fi

    # Copy service files
    for unit_file in "$systemd_source"/*.{service,timer}; do
        if [[ -f "$unit_file" ]]; then
            local unit_name=$(basename "$unit_file")
            cp "$unit_file" /etc/systemd/system/
            log_info "✓ Deployed $unit_name"
        fi
    done

    # Reload systemd
    systemctl daemon-reload
    log_info "✓ Reloaded systemd daemon"

    # Enable and start agent service
    if systemctl is-enabled --quiet aistack-agent.service 2>/dev/null; then
        log_info "✓ aistack-agent.service already enabled"
    else
        systemctl enable aistack-agent.service
        log_info "✓ Enabled aistack-agent.service"
    fi

    # Start the service if not running
    if systemctl is-active --quiet aistack-agent.service; then
        log_info "✓ aistack-agent.service is already running"
    else
        systemctl start aistack-agent.service
        sleep 1

        if systemctl is-active --quiet aistack-agent.service; then
            log_info "✓ Started aistack-agent.service"
        else
            log_error "Failed to start aistack-agent.service"
            systemctl status aistack-agent.service --no-pager || true
            exit 1
        fi
    fi

    # Enable timer (but don't start yet - placeholder)
    if [[ -f /etc/systemd/system/aistack-idle.timer ]]; then
        systemctl enable aistack-idle.timer
        log_info "✓ Enabled aistack-idle.timer (will activate after reboot)"
    fi
}

# Deploy logrotate configuration
deploy_logrotate() {
    log_info "Deploying logrotate configuration..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local logrotate_source="${script_dir}/assets/logrotate/aistack"

    if [[ ! -f "$logrotate_source" ]]; then
        log_error "logrotate config not found: $logrotate_source"
        exit 1
    fi

    cp "$logrotate_source" /etc/logrotate.d/aistack
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
    echo "  aistack Bootstrap Installer (EP-002)"
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
    deploy_systemd_units
    deploy_logrotate
    deploy_udev_rules

    echo ""
    log_info "========================================="
    log_info "Bootstrap completed successfully!"
    log_info "========================================="
    log_info ""
    log_info "Next steps:"
    log_info "  1. Check service status: systemctl status aistack-agent"
    log_info "  2. View logs: journalctl -u aistack-agent -f"
    log_info "  3. Run aistack CLI: aistack --help"
    log_info ""
    log_info "Bootstrap log: /tmp/aistack-bootstrap.log"
}

# Run main function
main "$@"
