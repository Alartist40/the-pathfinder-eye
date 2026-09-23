# AntiDoom for Pathfinder-Eye — Ministral-3B Loop Reduction

AntiDoom eliminates repetition loops ("doom loops") in the Ministral-3B
conversation model used in the Pathfinder-Eye's "Attention" / thinking mode.

Based on Liquid AI's AntiDoom pipeline — generates targeted preference pairs
by sampling model completions, detecting where repetition begins, marking
the loop-starting token as rejected, choosing coherent alternatives from the
model's own logprobs at that position, and training a LoRA adapter via
Final Token Preference Optimization (FTPO).

## Where it fits in the robot

```
User voice → Whisper STT → Needle 2 (Tool / Intent) → dispatchAction
                                                      ↓
                                              (if "Attention")
                                                      ↓
                                          startAIConversationLoop
                                                      ↓
                                          AIBrain.Process(text, context)
                                                      ↓
                                          leafcutter.service (systemd)
                                                      ↓
                                          Ministral-3B ← AntiDoom LoRA here
```

AntiDoom touches ONLY the Ministral model file on disk. Zero code changes
to the Go brain, leafcutter, or the systemd unit. It's a model-level fix.

## Files

```
antidoom/
├── config.yaml            — AntiDoom config tuned for Ministral-3B
├── build_prompts.py       — extracts conversation transcripts from dendrite or leafcutter log
├── AntiDoom_Training_Colab.ipynb  — one-click Colab training
├── deploy_antidoom.sh              — Pi deployment script (run on the robot)
└── README.md                      — this file
```

## Workflow

### 1. Gather prompts (run on the Pi, before Colab):

```bash
# Extract conversation transcripts from the dendrite logs
python antidoom/build_prompts.py \
    --dendrite /home/pi/the-pathfinder-eye_ai/db/dendrite.sqlite \
    --output antidoom/conversation_prompts.jsonl \
    --min-turns 3 \
    --max-prompt-len 1500

# Or from the leafcutter log directly:
python antidoom/build_prompts.py \
    --leafcutter-log /home/pi/the-pathfinder-eye_ai/logs/leafcutter.log \
    --output antidoom/conversation_prompts.jsonl

# If no real data exists: synthetic fallback prompts are auto-generated
# (the script emits 50 template prompts about camporee/robot topics)
```

### 2. Train on Colab (~30 min on T4):

Upload `config.yaml` + `conversation_prompts.jsonl` + the AntiDoom repo to Colab.
Run the notebook in order:
  → Generate FTPO preference pairs (model sampled at temp 0.01 → detector finds loops → rejected token + chosen alternatives)
  → Train LoRA adapter (FTPO + MSE tether)
  → Merge adapter into base model
  → Convert to GGUF for LeafcutterLLM

### 3. Deploy to the Pi:

```bash
# From your laptop:
scp ministral-3b-antidoom.gguf pi@robot:~/the-pathfinder-eye_ai/models/

# On the Pi:
chmod +x antidoom/deploy_antidoom.sh
./antidoom/deploy_antidoom.sh ministral-3b-antidoom.gguf
```

The script backs up the current model, stops leafcutter, updates the
systemd unit, restarts, and verifies.

### 4. Test:

Say "Attention" to activate the AI conversation loop. Have a long
conversation — the model should no longer enter repetition loops.

## AntiDoom config notes (tuned for Ministral-3B)

| Setting | Default AntiDoom | Pathfinder-Eye tuning | Why |
|---------|-----------------|----------------------|-----|
| target_pairs | 20000 | 500 | 3B model loops less; fewer pairs needed |
| max_seq_length | 6000 | 4096 | Ministral's context window |
| gpu_memory_utilization | 0.85 | 0.75 | Leave room on modest GPU |
| max_train_examples | 12000 | 300 | Proportionate to pair count |
| lora_r | 256 | 128 | Smaller model, smaller adapter |
| learning_rate | 0.000015 | 0.00002 | Tiny model learns faster, needs slightly higher LR |
| min_repeats | 4 | 3 | Smaller models produce shorter loops; catch earlier |
| min_total_repeated | 60 | 40 | Same reason |
| batch_size | 4 | 2 | Fits on T4 16GB for 3B model |

## How to revert

The deploy script automatically backs up the original model. Revert:

```bash
sudo systemctl stop leafcutter
sudo sed -i "s|--model .*\.gguf|--model Ministral-3-3B-Reasoning-2512-Q4_K_M.gguf|" \
    /etc/systemd/system/leafcutter.service
sudo systemctl daemon-reload && sudo systemctl start leafcutter
```

Or just swap the model file back and restart leafcutter via the Go brain's
`SwapLeafcutterModel()` function (leafcutter_swap.go).

## Why AntiDoom instead of just using a different model?

- **It's targeted.** AntiDoom doesn't change anything about Ministral's
  reasoning — it only suppresses the specific token preference that
  triggers loops. The model stays good at the things Ministral is good at.
- **It's cheap.** One Colab session. No multi-week fine-tuning.
- **It's model-agnostic.** If you switch to a Qwen or Llama model later,
  run the same pipeline on the new model's checkpoint.
- **It learns from your real data.** If you feed it the conversation
  transcripts where the robot actually entered a loop, it trains
  specifically on the contexts your users trigger.