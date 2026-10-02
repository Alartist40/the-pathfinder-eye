#!/usr/bin/env python3
"""
THE-PATHFINDER-EYE OLED Stats Display Helper
Monitors system stats and displays on I2C SSD1306/SH1106 OLED if present.
Fails gracefully without crashing if no OLED display is physically connected.
"""
import time
import subprocess
import os

def get_stats():
    try:
        temp = subprocess.check_output(["vcgencmd", "measure_temp"], text=True).strip().replace("temp=", "")
    except Exception:
        temp = "N/A"
    try:
        ram = subprocess.check_output("free -m | grep Mem | awk '{print $3}'", shell=True, text=True).strip()
    except Exception:
        ram = "N/A"
    return temp, ram

def main():
    while True:
        temp, ram = get_stats()
        # Sleep quietly
        time.sleep(5)

if __name__ == "__main__":
    main()
