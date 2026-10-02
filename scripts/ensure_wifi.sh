#!/bin/bash
# THE-PATHFINDER-EYE WiFi Auto-Connect & Offline Hotspot AP Manager

HOTSPOT_NAME="Pathfinder-Eye-Hotspot"
HOTSPOT_SSID="Pathfinder-Eye"

# Function to check if WiFi interface has an IP address
has_ip() {
    ip -4 addr show wlan0 2>/dev/null | grep -q "inet "
}

# Function to create and start the Hotspot AP
start_hotspot() {
    echo "[$(date)] Starting offline Hotspot: $HOTSPOT_SSID..."
    
    # Check if hotspot connection profile already exists
    if ! nmcli connection show "$HOTSPOT_NAME" >/dev/null 2>&1; then
        nmcli connection add type wifi ifname wlan0 con-name "$HOTSPOT_NAME" autoconnect no ssid "$HOTSPOT_SSID" 2>/dev/null || true
        nmcli connection modify "$HOTSPOT_NAME" 802-11-wireless.mode ap 802-11-wireless.band bg ipv4.method shared 2>/dev/null || true
    fi

    # Activate hotspot
    nmcli connection up "$HOTSPOT_NAME" 2>/dev/null || true
}

echo "[$(date)] WiFi Persistence & Hotspot Manager active."

# Initial startup delay to allow standard WiFi auto-connect
sleep 10

while true; do
    if ! has_ip; then
        echo "[$(date)] No active WiFi connection. Scanning for known networks..."
        nmcli device wifi rescan 2>/dev/null || true
        sleep 5
        
        # Check if known network connected during scan
        if ! has_ip; then
            echo "[$(date)] No known WiFi connected. Activating standalone Hotspot..."
            start_hotspot
        fi
    fi
    sleep 30
done
