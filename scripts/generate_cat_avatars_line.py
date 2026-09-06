#!/usr/bin/env python3
"""Generate minimalist line art cat avatars using Hugging Face Inference API.
Style: simple black line drawing on white background, cute rounded cat, minimal detail, doodle style."""

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

STYLE = "minimalist line art, simple black ink outline drawing on plain white background, cute rounded chubby cat, kawaii style, clean lines, no shading, no color, doodle sketch, simple and adorable, centered composition, white background"

CATS = [
    ("line-hacker", "cat wearing a tiny hoodie sitting at a laptop computer, focused expression"),
    ("line-zen", "cat sitting in meditation pose with closed eyes, peaceful content smile, tiny pink blush cheeks"),
    ("line-coder", "cat wearing round glasses looking at viewer, sitting with a small keyboard"),
    ("line-music", "cat wearing oversized headphones, eyes closed, happy expression, musical notes floating"),
    ("line-sleeping", "cat curled up in a ball sleeping peacefully on a keyboard, zzz"),
    ("line-rocket", "cat riding on top of a small rocket ship flying upward, eyes closed peacefully"),
    ("line-coffee", "cat holding a steaming coffee mug with both paws, sleepy morning expression"),
    ("line-ninja", "cat wearing a tiny ninja mask, sneaky pose, alert eyes"),
    ("line-wizard", "cat wearing a pointy wizard hat with a small star, magical sparkles around"),
    ("line-pirate", "cat with a tiny eyepatch and bandana, cheeky grin"),
    ("line-detective", "cat wearing a deerstalker hat holding a magnifying glass, curious expression"),
    ("line-dj", "cat with headphones around neck standing behind turntables, cool pose"),
]

print(f"🐱 Generating {len(CATS)} line art cat avatars...")
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
    if f.startswith("cat-line") and (f.endswith(".png") or f.endswith(".jpg")):
        size = os.path.getsize(os.path.join(OUTPUT_DIR, f))
        print(f"  {f} ({size/1024:.0f} KB)")
