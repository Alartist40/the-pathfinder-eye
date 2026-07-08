#!/usr/bin/env python3
"""
Compute route centroids for semantic_router.go
mirrors the Go sentenceEmbedding and normalize functions.
"""

import math
import re

VOCAB_SIZE = 256

def hash_string(s):
    h = 5381
    for c in s:
        h = ((h << 5) + h) + ord(c)
    if h < 0:
        h = -h
    return h

def normalize(vec):
    mag = math.sqrt(sum(x * x for x in vec))
    if mag == 0:
        return vec
    return [x / mag for x in vec]

def sentence_embedding(text, vocab_size=VOCAB_SIZE):
    words = text.lower().split()
    n = len(words)
    if n == 0:
        return [0.0] * vocab_size

    vec = [0.0] * vocab_size
    for i, word in enumerate(words):
        pos_weight = 1.0 - (i / n)
        h = hash_string(word)
        dim1 = h % vocab_size
        dim2 = (h >> 8) % vocab_size
        vec[dim1] += 0.7 * pos_weight
        vec[dim2] += 0.3 * pos_weight

    return normalize(vec)

def centroid(embeddings):
    n = len(embeddings[0])
    c = [0.0] * n
    for emb in embeddings:
        for j in range(n):
            c[j] += emb[j]
    return normalize([x / len(embeddings) for x in c])

# ─── Expanded training utterances ───────────────────────────────────────────

BASIC_CHAT = """
hello hi hey good morning good evening good afternoon good night
thanks thank you bye ok sure cool nice to meet you whats up
how are you howdy yo greetings sup namaste good day welcome
hey there hi there yo yo hey buddy gday cheers alright
""".split()

THINKING = """
why how would you compare analyze what are the pros and cons
walk me through the logic what causes summarize explain
what do you think about reason through thinking
break down elaborate on examine investigate
what is the difference between define clarify
tell me more give me details insights
""".split()

TOOL_CALL = """
# Robot-specific commands - the key expansion for STT robustness
birdwatch mode third watch bired wach beard watch word watch
birdwatch activate birdwatch enable
pathfinder song adventurer song play song play music
pathfinder law pathfinder pledge pathfinder aim pathfinder motto
adventurer law adventurer pledge adventurer aim
read the pathfinder law read the pledge read the aim read the motto
move forward go forward advance forward move ahead
move back go back reverse backward retreat
turn left spin left pivot left rotate left
turn right spin right pivot right rotate right
look up gaze up tilt up raise head
look down gaze down tilt down lower head
look center gaze center neutral position
look left gaze left pan left
look right gaze right pan right
follow mode track target start following activate follow
stop following disable follow exit follow mode
security mode activate security enable cameras turn on pir sensor
disable security turn off cameras deactivate security
deep thought thinking mode suspend vision enter deep thought
attention activate ai start ai conversation wake up pathfinder
remote control mode enable rc control
diagnostic test run hardware check test motors test servos
translate japanese mode start translation
stop exit sleep power down shut down
turn about about turn rotate around U-turn
move servo pan left pan right tilt up tilt down center camera
test led test motors run system check
scan network web search weather stock
network scan find devices on lan
get weather temperature forecast
play pathfinder soundtrack play adventurer music
"""

# Split on whitespace but keep multi-word phrases together
import re as re_module

def smart_split(text):
    # Keep "birdwatch mode", "turn about" etc as single tokens by protecting them
    protected = {
        "birdwatch mode": "BIRDWATCHMODE",
        "third watch": "THIRDWATCH",
        "bired wach": "BIREDWACH",
        "beard watch": "BEARDWATCH",
        "word watch": "WORDWATCH",
        "pathfinder song": "PATHFINDERSONG",
        "adventurer song": "ADVENTURERSONG",
        "pathfinder law": "PATHFINDERLAW",
        "pathfinder pledge": "PATHFINDERPLEDGE",
        "pathfinder aim": "PATHFINDERAIM",
        "pathfinder motto": "PATHFINDERMOTTO",
        "adventurer law": "ADVENTURERLAW",
        "adventurer pledge": "ADVENTURERPLEDGE",
        "adventurer aim": "ADVENTURERAIM",
        "adventurer motto": "ADVENTURERMOTTO",
        "move forward": "MOVEFORWARD",
        "go forward": "GOFORWARD",
        "move back": "MOVEBACK",
        "go back": "GOBACK",
        "move backward": "MOVEBACKWARD",
        "turn left": "TURNLEFT",
        "spin left": "SPINLEFT",
        "turn right": "TURNRIGHT",
        "spin right": "SPINRIGHT",
        "look up": "LOOKUP",
        "look down": "LOOKDOWN",
        "look center": "LOOKCENTER",
        "look left": "LOOKLEFT",
        "look right": "LOOKRIGHT",
        "gaze up": "GAZEUP",
        "gaze down": "GAZEDOWN",
        "gaze center": "GAZECENTER",
        "gaze left": "GAZELEFT",
        "gaze right": "GAZERIGHT",
        "follow mode": "FOLLOWMODE",
        "track target": "TRACKTARGET",
        "start following": "STARTFOLLOWING",
        "security mode": "SECURITYMODE",
        "turn on pir": "TURNONPIR",
        "activate follow": "ACTIVATERFOLLOW",
        "thinking mode": "THINKINGMODE",
        "enter deep": "ENTERDEEP",
        "remote control": "REMOTECONTROL",
        "diagnostic test": "DIAGNOSTIC",
        "run hardware": "RUNHARDWARE",
        "test motors": "TESTMOTORS",
        "test servos": "TESTSERVOS",
        "test led": "TESTLED",
        "run system": "RUNSYSTEM",
        "scan network": "SCANNETWORK",
        "web search": "WEBSEARCH",
        "get weather": "GETWEATHER",
        "turn about": "TURNABOUT",
        "about turn": "ABOUTTURN",
        "turn on pir sensor": "TURNONPIRSENSOR",
        "activate security": "ACTIVATESECURITY",
        "activate ai": "ACTIVATEAI",
        "wake up": "WAKEUP",
        "power down": "POWERDOWN",
        "shut down": "SHUTDOWN",
        "stop following": "STOPFOLLOWING",
        "disable follow": "DISABLEFOLLOW",
        "exit follow": "EXITFOLLOW",
        "deactivate security": "DEACTIVATESECURITY",
        "turn off cameras": "TURNOFFCAMS",
        "disable security": "DISABLESECURITY",
        "suspend vision": "SUSPENDVISION",
        "start translation": "STARTTRANSLATION",
        "japanese mode": "JAPANESEMODE",
        "translate japanese": "TRANSLATEJAPANESE",
        "U-turn": "UTURN",
        "deep thought": "DEEPTHOUGHT",
    }
    
    t = text
    for phrase, token in protected.items():
        t = t.replace(phrase, token)
    
    words = t.split()
    
    result = []
    for w in words:
        if w in protected.values():
            # Map back to spaced version for embedding (treat as single pseudo-word)
            for phrase, token in protected.items():
                if token == w:
                    # Split into individual words for embedding (embedding handles it)
                    result.extend(phrase.split())
                    break
        else:
            result.append(w)
    return result

def compute_centroid_for_phrases(phrases_text):
    embeddings = []
    for phrase in phrases_text.strip().split('\n'):
        phrase = phrase.strip()
        if not phrase:
            continue
        words = smart_split(phrase)
        emb = sentence_embedding(' '.join(words))
        embeddings.append(emb)
    return centroid(embeddings)

def embedding_to_go(emb, name):
    # Format as Go array literal (8 per line)
    lines = []
    for i in range(0, len(emb), 8):
        chunk = emb[i:i+8]
        vals = ', '.join(f'{v:.3f}' for v in chunk)
        lines.append(f'\t{vals},')
    return f'var {name} = Embedding{{\n' + '\n'.join(lines) + '\n}'

basic_emb = [sentence_embedding(' '.join(BASIC_CHAT))]
basic_c = centroid(basic_emb)

thinking_emb = [sentence_embedding(' '.join(THINKING))]
thinking_c = centroid(thinking_emb)

# Compute tool_call centroid from expanded data
tool_phrases = TOOL_CALL.strip().split('\n')
tool_embeddings = []
for phrase in tool_phrases:
    phrase = phrase.strip()
    if not phrase:
        continue
    words = smart_split(phrase)
    emb = sentence_embedding(' '.join(words))
    tool_embeddings.append(emb)

tool_c = centroid(tool_embeddings)

print(f'Basic chat centroid: {len(basic_c)} dims')
print(f'Thinking centroid: {len(thinking_c)} dims')
print(f'Tool call centroid: {len(tool_c)} dims')
print()

# Print as Go code
print(embedding_to_go(basic_c, 'basicChatCentroid'))
print()
print(embedding_to_go(thinking_c, 'thinkingCentroid'))
print()
print(embedding_to_go(tool_c, 'toolCallCentroid'))