#!/bin/bash
# Enable Wake-on-LAN for all Ethernet interfaces
# This script is called by systemd to persist WoL settings

set -euo pipefail

# Log function for systemd journal
log() {
    echo "$1"
    logger -t aistack-wol "$1"
}

# Find all active Ethernet interfaces (exclude loopback, virtual, wifi)
find_ethernet_interfaces() {
    local interfaces=()

    for iface in /sys/class/net/*; do
        iface=$(basename "$iface")

        # Skip loopback
        [[ "$iface" == "lo" ]] && continue

        # Skip virtual interfaces (docker, etc)
        [[ "$iface" =~ ^(docker|br-|veth|virbr) ]] && continue

        # Skip WiFi (wlan, wlp)
        [[ "$iface" =~ ^(wlan|wlp) ]] && continue

        # Check if interface is up or has a carrier
        if [[ -e "/sys/class/net/$iface/carrier" ]] || \
           [[ -e "/sys/class/net/$iface/operstate" ]]; then
            interfaces+=("$iface")
        fi
    done

    echo "${interfaces[@]}"
}

# Enable WoL for a specific interface
enable_wol_for_interface() {
    local iface=$1

    # Check if ethtool is available
    if ! command -v ethtool &> /dev/null; then
        log "ERROR: ethtool not found, cannot enable WoL"
        return 1
    fi

    # Check if interface supports WoL
    if ! ethtool "$iface" &> /dev/null; then
        log "SKIP: Cannot query $iface with ethtool"
        return 0
    fi

    # Check WoL support
    local supports_wol=$(ethtool "$iface" 2>/dev/null | grep "Supports Wake-on:" | awk '{print $3}')
    if [[ -z "$supports_wol" ]] || [[ "$supports_wol" == "d" ]]; then
        log "SKIP: $iface does not support Wake-on-LAN"
        return 0
    fi

    # Check current WoL setting
    local current_wol=$(ethtool "$iface" 2>/dev/null | grep "Wake-on:" | tail -1 | awk '{print $2}')

    if [[ "$current_wol" == "g" ]]; then
        log "OK: $iface already has Wake-on-LAN enabled"
        return 0
    fi

    # Enable WoL
    if ethtool -s "$iface" wol g 2>/dev/null; then
        log "ENABLED: Wake-on-LAN for $iface (was: $current_wol)"
    else
        log "ERROR: Failed to enable Wake-on-LAN for $iface"
        return 1
    fi
}

# Main logic
main() {
    log "Starting Wake-on-LAN persistence service"

    # Find all Ethernet interfaces
    interfaces=$(find_ethernet_interfaces)

    if [[ -z "$interfaces" ]]; then
        log "WARNING: No Ethernet interfaces found"
        exit 0
    fi

    log "Found interfaces: $interfaces"

    # Enable WoL for each interface
    for iface in $interfaces; do
        enable_wol_for_interface "$iface"
    done

    log "Wake-on-LAN persistence service completed"
}

main "$@"
