#!/usr/bin/env python3
"""Generate Salvador Dali surrealist cat avatars using Hugging Face Inference API.
Style: melting clocks, surrealist landscapes, dreamlike, oil painting, Dali-inspired."""

import os
import sys
import time
import json
import urllib.request

HF_TOKEN = os.environ.get("HF_TOKEN")
if not HF_TOKEN:
    print("ERROR: HF_TOKEN env var required")
    sys.exit(1)

OUTPUT_DIR = os.path.expanduser("~/projects/wee/internal/server/frontend/public/avatars/cats")
MODEL = "black-forest-labs/FLUX.1-schnell"
API_URL = f"https://router.huggingface.co/hf-inference/models/{MODEL}"

os.makedirs(OUTPUT_DIR, exist_ok=True)

STYLE = "Salvador Dali surrealist oil painting style, melting dripping forms, dreamlike surreal landscape, long stretched shadows, desert horizon, warm golden amber tones, hyper-detailed oil paint texture, persistence of memory aesthetic, floating impossible geometry, fine art masterpiece"

CATS = [
    ("dali-hacker", "a surreal cat in a hoodie with melting laptop screens dripping like clocks, glowing code floating in the desert sky"),
    ("dali-zen", "a serene cat meditating floating above a surreal desert, body melting softly into the sand, third eye glowing"),
    ("dali-coder", "a cat with impossibly long stretched legs wearing round glasses, typing on a keyboard made of melting keys"),
    ("dali-music", "a cat with enormous melting headphones, musical notes dripping like liquid gold across a barren dreamscape"),
    ("dali-sleeping", "a cat sleeping on a melting clock draped over a tree branch, soft dreamlike desert background"),
    ("dali-rocket", "a cat riding a melting rocket ship through a surreal sky with floating eyes and impossible staircases"),
    ("dali-coffee", "a cat holding a coffee cup that melts and drips upward defying gravity, steam forming surreal shapes"),
    ("dali-ninja", "a surreal ninja cat with elongated shadow, multiple reflections in floating mirrors across desert landscape"),
    ("dali-wizard", "a cat wizard with a melting pointed hat, casting spells that drip like liquid paint, elephants on stilts in background"),
    ("dali-pirate", "a pirate cat with an eyepatch standing on a melting ship, ocean made of liquid clocks and golden light"),
    ("dali-detective", "a cat detective with a magnifying glass that distorts reality, warped buildings and stretched perspectives"),
    ("dali-dj", "a cat behind melting turntables, vinyl records drooping like soft clocks, sound waves visible as dripping ribbons"),
]

print(f"🎨 Generating {len(CATS)} Salvador Dali cat avatars...")
print(f"   Model: {MODEL}")
print(f"   Output: {OUTPUT_DIR}")
print()

generated = 0
for name, desc in CATS:
    outfile = os.path.join(OUTPUT_DIR, f"cat-{name}.jpg")

    if os.path.exists(outfile):
        print(f"⏭️  Skipping cat-{name}.jpg (already exists)")
        generated += 1
        continue

    prompt = f"{desc}, {STYLE}"
    print(f"🎨 Generating cat-{name}.jpg ... ", end="", flush=True)

    payload = json.dumps({
        "inputs": prompt,
        "parameters": {
            "width": 512,
            "height": 512,
            "num_inference_steps": 4
        }
    }).encode("utf-8")

    req = urllib.request.Request(
        API_URL,
        data=payload,
        headers={
            "Authorization": f"Bearer {HF_TOKEN}",
            "Content-Type": "application/json",
        },
    )

    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            data = resp.read()
            if data[:4] == b'\x89PNG' or data[:3] == b'\xff\xd8\xff':
                with open(outfile, "wb") as f:
                    f.write(data)
                size_kb = len(data) / 1024
                print(f"✅ Done ({size_kb:.0f} KB)")
                generated += 1
            else:
                print(f"❌ Response was not an image")
                try:
                    print(f"   {json.loads(data)}")
                except:
                    print(f"   {data[:200]}")
    except Exception as e:
        print(f"❌ {e}")

    time.sleep(1)

print()
print(f"🎉 Generation complete! {generated}/{len(CATS)} avatars generated.")
for f in sorted(os.listdir(OUTPUT_DIR)):
    if f.startswith("cat-dali") and (f.endswith(".png") or f.endswith(".jpg")):
        size = os.path.getsize(os.path.join(OUTPUT_DIR, f))
        print(f"  {f} ({size/1024:.0f} KB)")
