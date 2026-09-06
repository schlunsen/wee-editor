#!/usr/bin/env python3
"""Generate cyberpunk cat avatars using Hugging Face Inference API.
Style matches existing wee.cat brand: neon magenta/cyan, dark background, digital art."""

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

STYLE = "cyberpunk neon cat portrait, dark black background, neon magenta and cyan and purple lighting, digital art, avatar icon, centered face, high detail, round composition, glowing neon accents"

CATS = [
    ("hacker", "cat wearing a hoodie with one eye showing through matrix-green sunglasses, mysterious hacker vibe"),
    ("zen", "serene cat meditating with closed eyes, peaceful expression, floating lotus pose"),
    ("coder", "cat wearing round glasses looking intensely focused at viewer, determined expression, coding programmer vibe"),
    ("music", "cat wearing large over-ear headphones, eyes closed, enjoying music, content smile"),
    ("sleeping", "cute sleeping cat curled up on a keyboard, peaceful dreamy expression, soft glow"),
    ("rocket", "adventurous cat in a small space helmet, excited expression, stars in background"),
    ("coffee", "cat holding a steaming coffee mug, sleepy morning expression, cozy vibe"),
    ("ninja", "stealthy ninja cat with a dark mask, sharp alert eyes, mysterious"),
    ("wizard", "cat wearing a small wizard hat with stars, mystical glowing eyes, magical sparkles"),
    ("pirate", "cat with an eyepatch and small bandana, mischievous grin, adventurous"),
    ("detective", "cat wearing a deerstalker hat and monocle, investigating expression, sherlock vibe"),
    ("dj", "cat with DJ headphones around neck, cool confident expression, turntable glow"),
]

print(f"🐱 Generating {len(CATS)} cat avatars...")
print(f"   Model: {MODEL}")
print(f"   Output: {OUTPUT_DIR}")
print()

generated = 0
for name, desc in CATS:
    outfile = os.path.join(OUTPUT_DIR, f"cat-{name}.jpg")

    if os.path.exists(outfile):
        print(f"⏭️  Skipping cat-{name}.png (already exists)")
        generated += 1
        continue

    prompt = f"{desc}, {STYLE}"
    print(f"🎨 Generating cat-{name}.png ... ", end="", flush=True)

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
            # Check if response is an image (PNG starts with \x89PNG)
            if data[:4] == b'\x89PNG' or data[:3] == b'\xff\xd8\xff':
                with open(outfile, "wb") as f:
                    f.write(data)
                size_kb = len(data) / 1024
                print(f"✅ Done ({size_kb:.0f} KB)")
                generated += 1
            else:
                # Might be JSON error
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
    if f.startswith("cat-") and (f.endswith(".png") or f.endswith(".jpg")):
        size = os.path.getsize(os.path.join(OUTPUT_DIR, f))
        print(f"  {f} ({size/1024:.0f} KB)")
