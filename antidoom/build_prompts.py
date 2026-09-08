#!/usr/bin/env python3
"""
Build a prompt dataset for AntiDoom from the Pathfinder-Eye's conversation logs.

Reads the dendrite DB's conversation transcript records and converts them
into a JSONL file that AntiDoom can use as its prompt source. Each row is
the conversation up to a point where the Ministral model is about to think
— the exact kind of prompt that would trigger a doom loop if the model is
prone to it.

Usage:
    # On the dev machine (with a copy of the Pi's logs):
    python pathfinder-eye/antidoom/build_prompts.py \
        --dendrite /home/xander/Documents/portfolio/the-pathfinder-eye/db/dendrite.sqlite \
        --output pathfinder-eye/antidoom/conversation_prompts.jsonl \
        --min-turns 3 \
        --max-prompt-len 1500

    # Or scrape from the leafcutter log directly:
    python antidoom/build_prompts.py \
        --leafcutter-log ~/leafcutter_conversations.log \
        --output conversation_prompts.jsonl
"""

import argparse
import json
import re
import sqlite3
import sys
from pathlib import Path


def extract_from_dendrite(dendrite_path, min_turns=3, max_prompt_chars=1500):
    """
    Extract conversation transcripts from the DENDRITE SQLite database.
    The dendrite stores conversation turns as Event nodes with
    [[links]] to their preceding turns.
    """
    prompts = []
    if not Path(dendrite_path).exists():
        print(f"ERROR: dendrite db not found at {dendrite_path}")
        print("Falling back to manual prompt mode — see readme.")
        return []

    conn = sqlite3.connect(dendrite_path)
    cur = conn.cursor()

    # Dendrite stores conversations as linked Event nodes.
    # Try the standard Cynapse schema: events table with type='conversation_turn'
    try:
        cur.execute("""
            SELECT node_id, body, created_at
            FROM events
            WHERE type = 'conversation_turn'
            ORDER BY created_at ASC
        """)
    except sqlite3.OperationalError:
        # Fallback: scan all text-bearing rows
        cur.execute("""
            SELECT rowid, body, created_at
            FROM events
            ORDER BY created_at ASC
        """)

    rows = cur.fetchall()
    conn.close()

    if not rows:
        print("No conversation turns found in dendrite.")
        return []

    # Group turns into conversation windows
    # (simplified: guess conversation boundaries by time gaps > 60s)
    conversations = []
    current_conv = []
    last_ts = None

    for node_id, body, ts in rows:
        if not body or not isinstance(body, str):
            continue
        if last_ts is not None and (ts - last_ts) > 60:
            if len(current_conv) >= min_turns:
                conversations.append(current_conv)
            current_conv = []
        current_conv.append(body)
        last_ts = ts

    if len(current_conv) >= min_turns:
        conversations.append(current_conv)

    # Convert to AntiDoom prompt format: each conversation up to the
    # last user turn (the model hasn't responded yet)
    for conv in conversations:
        # Build the conversation transcript
        transcript = ""
        for turn in conv:
            transcript += turn.strip() + "\n"

        # Trim to max prompt length
        if len(transcript) > max_prompt_chars:
            transcript = transcript[-max_prompt_chars:]

        prompts.append({
            "prompt": transcript.strip(),
            "source": "pathfinder-eye-dendrite",
            "conversation_length": len(conv),
        })

    return prompts


def extract_from_leafcutter_log(log_path, min_turns=3, max_prompt_chars=1500):
    """
    Extract conversation transcripts from the leafcutter server log.
    The log format is flat-text with user/assistant turns separated by
    known markers (e.g. "USER:" / "ASSISTANT:").
    """
    prompts = []
    if not Path(log_path).exists():
        print(f"ERROR: log not found at {log_path}")
        return []

    with open(log_path, "r", encoding="utf-8", errors="replace") as f:
        text = f.read()

    # Find conversation turns — heuristic: look for pairs of
    # "USER:" and "ASSISTANT:" lines
    turns = re.findall(
        r'(?:USER:\s*(.+?)\s*ASSISTANT:\s*(.+?))(?=USER:|$)',
        text, re.DOTALL | re.IGNORECASE
    )

    conversations = []
    current_conv = []

    for user_turn, assistant_turn in turns:
        current_conv.append(user_turn.strip())
        current_conv.append(assistant_turn.strip())

        # A conversation is a sequence of alternating turns
        if len(current_conv) >= 6:  # at least 3 exchanges
            # Build prompt: all turns up to the last user message
            # (we want the model to think, not replay its previous answer)
            prompt_turns = current_conv[:-1]  # drop the last assistant turn
            transcript = "\n".join(prompt_turns)

            if len(transcript) > max_prompt_chars:
                transcript = transcript[-max_prompt_chars:]

            conversations.append({
                "prompt": transcript.strip(),
                "source": "pathfinder-eye-leafcutter-log",
                "conversation_length": len(current_conv),
            })
            current_conv = []

    return conversations


def generate_fallback_prompts(count=50, max_prompt_chars=1500):
    """
    If no dendrite or leafcutter data is available, generate synthetic
    conversation prompts that would realistically trigger doom loops in
    a reasoning model. These are seed prompts — AntiDoom generates actual
    completions from the model, then detects loops in those.
    """
    templates = [
        # Long reasoning prompts that trigger chain-of-thought loops
        "A camper asks: {question}. The robot has access to these tools: {tools}. Think step by step.",
        "The user just said: \"{user_text}\". Previous conversation: {history}. Respond and explain your reasoning.",
        "Pathfinder Eye has detected: {detection}. The user wants to know: {request}. What would you say?",
    ]

    questions = [
        "How far can you see with your cameras? What's the range?",
        "I see a bird up in that tree. Can you identify what species it is?",
        "What do you know about the history of this trail?",
        "Describe everything you can see right now, in detail.",
        "What's the fastest way to navigate through rough terrain?",
        "What's the difference between a pathfinder and an adventurer?",
        "Tell me about the camporee traditions.",
        "If I give you a set of coordinates, can you find them on a map?",
    ]

    tools_desc = "camera (YOLOv5 object detection), gimbal (pan/tilt), motors (4WD), speaker (espeak-ng), microphone (Whisper STT)"

    prompts = []
    for i in range(count):
        q = questions[i % len(questions)]
        t = templates[i % len(templates)]
        text = t.format(
            question=q,
            tools=tools_desc,
            user_text=q,
            history="(earlier conversation turns go here)",
            detection="a person 2m ahead, bird in tree to left",
            request=q,
        )
        if len(text) > max_prompt_chars:
            text = text[:max_prompt_chars]
        prompts.append({
            "prompt": text,
            "source": "pathfinder-eye-synthetic",
        })

    return prompts


def main():
    parser = argparse.ArgumentParser(
        description="Build AntiDoom prompt dataset from pathfinder-eye conversation data"
    )
    parser.add_argument("--dendrite", type=str, help="Path to dendrite.sqlite")
    parser.add_argument("--leafcutter-log", type=str, help="Path to leafcutter conversation log")
    parser.add_argument("--output", type=str, required=True, help="Output JSONL path")
    parser.add_argument("--min-turns", type=int, default=3)
    parser.add_argument("--max-prompt-len", type=int, default=1500)
    parser.add_argument("--fallback-count", type=int, default=50,
                        help="Number of synthetic prompts if no real data")
    args = parser.parse_args()

    prompts = []

    if args.dendrite:
        print(f"Extracting from dendrite: {args.dendrite}")
        prompts.extend(extract_from_dendrite(
            args.dendrite, args.min_turns, args.max_prompt_len
        ))

    if args.leafcutter_log:
        print(f"Extracting from leafcutter log: {args.leafcutter_log}")
        prompts.extend(extract_from_leafcutter_log(
            args.leafcutter_log, args.min_turns, args.max_prompt_len
        ))

    if not prompts:
        print("No real conversation data found. Generating synthetic fallback prompts.")
        prompts = generate_fallback_prompts(args.fallback_count, args.max_prompt_len)

    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    with open(output_path, "w") as f:
        for p in prompts:
            f.write(json.dumps(p, ensure_ascii=False) + "\n")

    print(f"Wrote {len(prompts)} prompts to {output_path}")
    print(f"Sources: {set(p.get('source', '?') for p in prompts)}")


if __name__ == "__main__":
    main()