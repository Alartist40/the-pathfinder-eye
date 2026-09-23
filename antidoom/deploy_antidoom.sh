#!/bin/bash
# THE-PATHFINDER-EYE : AntiDoom Deployment Script
#
# This runs ON THE PI, not on the dev workstation.
# It replaces the Ministral-3B GGUF model with an AntiDoom-ed version
# that has been trained to avoid doom loops in the conversation loop.
#
# Prerequisites:
#   - ministral-3b-antidoom.gguf already copied to the Pi (via scp from Colab)
#   - or: run the Colab notebook first, download the gguf, then scp it
#
# Usage:
#   chmod +x pathfinder-eye/antidoom/deploy_antidoom.sh
#   ./pathfinder-eye/antidoom/deploy_antidoom.sh [new_model_path]
#
# If no argument is given, defaults to:
#   /home/pi/the-pathfinder-eye_ai/models/ministral-3b-antidoom.gguf

set -e

MODELS_DIR="/home/pi/the-pathfinder-eye_ai/models"
OLD_MODEL="$MODELS_DIR/Ministral-3-3B-Reasoning-2512-Q4_K_M.gguf"
NEW_MODEL="${1:-$MODELS_DIR/ministral-3b-antidoom.gguf}"
BACKUP_DIR="$MODELS_DIR/backups"

echo "══════════════════════════════════════════════"
echo "AntiDoom Ministral-3B Deployment"
echo "══════════════════════════════════════════════"
echo ""

# --- 1. Verify the new model exists ---
if [[ ! -f "$NEW_MODEL" ]]; then
    echo "ERROR: New model not found at $NEW_MODEL"
    echo ""
    echo "Steps to get the model:"
    echo "  1. Run the AntiDoom Colab notebook (antidoom/AntiDoom_Training_Colab.ipynb)"
    echo "  2. Download ministral-3b-antidoom.gguf to your laptop"
    echo "  3. scp it to the Pi:"
    echo "     scp ministral-3b-antidoom.gguf pi@<pi-ip>:$MODELS_DIR/"
    exit 1
fi

NEW_SIZE=$(du -h "$NEW_MODEL" | cut -f1)
echo "[1/4] New model: $NEW_MODEL ($NEW_SIZE)"

# --- 2. Backup the current model ---
if [[ -f "$OLD_MODEL" ]]; then
    echo "[2/4] Backing up current model..."
    mkdir -p "$BACKUP_DIR"
    cp "$OLD_MODEL" "$BACKUP_DIR/$(basename "$OLD_MODEL").$(date +%Y%m%d_%H%M%S)"
    echo "  Backup saved to $BACKUP_DIR"
else
    echo "[2/4] No existing model to backup (first install)."
fi

# --- 3. Stop leafcutter, swap model, restart ---
echo "[3/4] Swapping leafcutter model..."

if systemctl is-active --quiet leafcutter 2>/dev/null; then
    echo "  Stopping leafcutter..."
    sudo systemctl stop leafcutter
    sleep 2
fi

# Update the canonical unit file to point at the new model
# (leafcutter_swap.go handles hot-swapping during operation;
#  this is the permanent switch on initial deployment)
LEAFCUTTER_UNIT="/etc/systemd/system/leafcutter.service"
if [[ -f "$LEAFCUTTER_UNIT" ]]; then
    sudo sed -i "s|--model .*\.gguf|--model $NEW_MODEL|" "$LEAFCUTTER_UNIT"
    sudo systemctl daemon-reload
    echo "  Unit file updated to: --model $NEW_MODEL"
else
    echo "  WARNING: leafcutter.service not found — model swap will be manual"
fi

# --- 4. Start and verify ---
echo "[4/4] Starting leafcutter with AntiDoom model..."
sudo systemctl start leafcutter
sleep 5

if systemctl is-active --quiet leafcutter; then
    echo ""
    echo "✅ SUCCESS — AntiDoom Ministral-3B is now serving."
    echo "   Model: $NEW_MODEL"
    echo ""
    echo "To revert to the original model:"
    echo "  sudo systemctl stop leafcutter"
    echo "  sudo sed -i 's|--model .*|--model $OLD_MODEL|' $LEAFCUTTER_UNIT"
    echo "  sudo systemctl daemon-reload && sudo systemctl start leafcutter"
else
    echo ""
    echo "⚠️  Leafcutter did not start. Check logs:"
    echo "   sudo journalctl -u leafcutter --no-pager -n 50"
    echo ""
    echo "Reverting to original model..."
    sudo sed -i "s|--model .*|--model $OLD_MODEL|" "$LEAFCUTTER_UNIT"
    sudo systemctl daemon-reload
    sudo systemctl start leafcutter
fi

echo ""
echo "══════════════════════════════════════════════"
echo "Done. Test with: say 'Attention' and have a long conversation."
echo "If the model still loops, re-run AntiDoom with more prompt pairs."
echo "══════════════════════════════════════════════"