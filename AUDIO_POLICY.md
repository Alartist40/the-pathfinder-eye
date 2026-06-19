# THE-PATHFINDER-EYE Absolute Audio Preservation Policy

## MANDATORY RULES - DO NOT OVERRIDE
1. **MIC SENSITIVITY:** Must remain at 100% with hardware 'Auto Gain Control' set to ON.
2. **RECORDING METHOD:** Only use fixed-length 'arecord' windows. NEVER use 'sox' or 'rec' VAD for the core interaction as it causes user frustration and deafness.
3. **WAKE WORD TIMING:** 3-second uninterruptible window.
4. **COMMAND TIMING:** 5-second uninterruptible window.
5. **HARDWARE MUTE:** Do NOT use software mute toggles on the microphone.

**PERMISSION REQUIRED:** Any changes to the core audio pipeline (/go_brain/voice.go) require explicit permission from the project owner (Xander). 
Violation of these settings will break the robot's ability to operate in noisy environments.
