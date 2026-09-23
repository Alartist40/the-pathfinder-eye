#!/usr/bin/env python3
"""
THE-PATHFINDER-EYE : Pocket-TTS Microservice
Pre-loads Kyutai Pocket-TTS model in RAM for instant, low-latency text-to-speech.
Exposes HTTP synthesis endpoint on port 8020.

Requires: pip install pocket-tts scipy
"""

import io
import json
import logging
import os
import sys
import threading
from http.server import HTTPServer, BaseHTTPRequestHandler

import numpy as np
import scipy.io.wavfile

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("pocket_tts_service")

tts_model = None
model_lock = threading.Lock()

# Default voice — change this to your cloned voice path (.safetensors) or a
# predefined voice name (alba, giovanni, lola, etc.)
DEFAULT_VOICE = os.environ.get("POCKET_TTS_VOICE", "alba")


class PocketTTSHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({
                "status": "healthy",
                "service": "pocket-tts",
                "model_loaded": tts_model is not None,
                "default_voice": DEFAULT_VOICE,
            }).encode())
        else:
            self.send_error(404, "Not Found")

    def do_POST(self):
        if self.path in ("/synthesize", "/tts", "/generate"):
            content_length = int(self.headers.get("Content-Length", 0))
            post_data = self.rfile.read(content_length)

            text = ""
            voice = DEFAULT_VOICE

            try:
                data = json.loads(post_data.decode("utf-8"))
                text = data.get("text", "")
                voice = data.get("voice", DEFAULT_VOICE)
            except Exception:
                text = post_data.decode("utf-8", errors="ignore")

            if not text.strip():
                self.send_error(400, "Empty text prompt")
                return

            logger.info(f"Synthesizing ({len(text)} chars): {text[:80]}...")

            try:
                wav_bytes = self.generate_wav(text, voice)
                self.send_response(200)
                self.send_header("Content-Type", "audio/wav")
                self.send_header("Content-Length", str(len(wav_bytes)))
                self.end_headers()
                self.wfile.write(wav_bytes)
            except Exception as err:
                logger.error(f"Synthesis error: {err}", exc_info=True)
                self.send_error(500, f"Synthesis failed: {err}")
        else:
            self.send_error(404, "Not Found")

    def log_message(self, format, *args):
        # Suppress default access logging to reduce noise
        pass

    def generate_wav(self, text: str, voice: str) -> bytes:
        global tts_model
        if tts_model is None:
            raise RuntimeError("Pocket-TTS model not loaded")

        with model_lock:
            # Resolve voice state — works with predefined names or .safetensors paths
            voice_state = tts_model.get_state_for_audio_prompt(voice)

            # Generate audio (non-streaming — Pocket TTS handles chunking internally)
            audio = tts_model.generate_audio(
                text_to_generate=text,
                model_state=voice_state,
            )

        # Convert to numpy
        if hasattr(audio, "numpy"):
            audio_np = audio.numpy()
        else:
            audio_np = np.array(audio)

        if audio_np.ndim > 1:
            audio_np = audio_np.squeeze()

        # Encode as WAV
        buf = io.BytesIO()
        sample_rate = getattr(tts_model.config.mimi, "sample_rate", 24000)
        scipy.io.wavfile.write(buf, sample_rate, (audio_np * 32767).astype(np.int16))
        return buf.getvalue()


def main():
    global tts_model

    logger.info("Pocket-TTS: Loading model into RAM...")
    try:
        from pocket_tts import TTSModel
        tts_model = TTSModel.load_model()
    except ImportError:
        # Fallback: try direct import if package isn't installed
        logger.warning("pocket_tts package not found via import, trying sys.path insert...")
        tts_dirs = [
            "/home/xander/Documents/reference/pocket-tts",
            "/home/xander/Documents/reference/pfe/pocket-tts",
        ]
        for d in tts_dirs:
            if os.path.isdir(d):
                sys.path.insert(0, d)
                break
        from pocket_tts import TTSModel
        tts_model = TTSModel.load_model()

    logger.info(f"Pocket-TTS: Model loaded. Default voice: {DEFAULT_VOICE}")
    logger.info("Pocket-TTS: Listening on port 8020...")

    server = HTTPServer(("0.0.0.0", 8020), PocketTTSHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("Pocket-TTS: Service stopping.")


if __name__ == "__main__":
    main()
