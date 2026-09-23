#!/bin/bash
# THE-PATHFINDER-EYE : Voice Cloning Setup
# Records a voice sample and exports it as a .safetensors voice state
# for instant loading by Pocket TTS.
#
# Usage:
#   ./setup_voice.sh                  # Interactive — records 30s sample
#   ./setup_voice.sh /path/to/voice.wav  # Uses existing audio file
#
# Requirements: arecord (alsa-utils), pocket-tts, scipy

set -e

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
VOICES_DIR="$PROJECT_DIR/voices"
SAMPLE_FILE="/tmp/voice_sample.wav"
OUTPUT_VOICE="$VOICES_DIR/custom_voice.safetensors"

mkdir -p "$VOICES_DIR"

echo "=== PATHFINDER EYE: Voice Cloning Setup ==="
echo ""

if [ -n "$1" ] && [ -f "$1" ]; then
    echo "Using provided audio: $1"
    SAMPLE_FILE="$1"
else
    echo "I will record a 30-second voice sample."
    echo "Speak clearly and naturally. Read a book, tell a story, anything."
    echo ""
    echo "Press Enter when ready to start recording..."
    read -r

    echo "Recording for 30 seconds... speak now!"
    arecord -D plughw:0,0 -d 30 -f S16_LE -r 24000 -c 1 "$SAMPLE_FILE"
    echo "Recording complete."
fi

echo ""
echo "Exporting voice to safetensors..."
python3 -c "
import sys
sys.path.insert(0, '$PROJECT_DIR')
from pocket_tts import TTSModel

model = TTSModel.load_model()
voice_state = model.export_voice_state('$SAMPLE_FILE')
model.save_voice_state(voice_state, '$OUTPUT_VOICE')
print(f'Voice saved to: $OUTPUT_VOICE')
print(f'Voice state size: {sum(v.numel() * v.element_size() for v in voice_state.values()) / 1024:.1f} KB')
"

echo ""
echo "=== Setup Complete ==="
echo ""
echo "To use your cloned voice:"
echo "  1. Set environment variable:  export POCKET_TTS_VOICE=$OUTPUT_VOICE"
echo "  2. Or update the systemd service:  POCKET_TTS_VOICE=$OUTPUT_VOICE"
echo ""
echo "To test: curl -X POST http://localhost:8020/tts -H 'Content-Type: application/json' -d '{\"text\":\"Hello world\",\"voice\":\"$OUTPUT_VOICE\"}' --output test.wav"
echo ""
echo "Available predefined voices: alba, giovanni, lola, juergen, rafael, estelle, anna, azelma"
