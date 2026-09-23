#!/bin/bash
# THE-PATHFINDER-EYE WiFi Persistence Script
while true; do
    if ! nmcli -t -f CONNECTIVITY connectivity | grep -q "full"; then
        echo "[$(date)] WiFi disconnected. Attempting to reconnect..."
        nmcli device wifi rescan
        sleep 5
        nmcli connection up "AiR-WiFi_0UQPGA" 2>/dev/null
    fi
    sleep 60
done
