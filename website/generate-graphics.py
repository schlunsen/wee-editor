"""Generate retro terminal-style graphics for the Wee presentation."""

import requests
import os
import time

API_URL = "https://router.huggingface.co/hf-inference/models/black-forest-labs/FLUX.1-schnell"
HF_API_KEY = os.environ.get("HF_API_KEY", "")

headers = {"Authorization": f"Bearer {HF_API_KEY}"}


def generate(prompt, filename, width=1280, height=480):
    """Generate an image from a prompt and save it."""
    if not HF_API_KEY:
        print(f"Skipping {filename} — set HF_API_KEY environment variable")
        return False

    print(f"Generating: {filename}")
    print(f"  Prompt: {prompt}")

    payload = {
        "inputs": prompt,
        "parameters": {
            "width": width,
            "height": height,
        },
    }

    for attempt in range(3):
        response = requests.post(API_URL, headers=headers, json=payload)
        if response.status_code == 200:
            with open(filename, "wb") as f:
                f.write(response.content)
            size_kb = len(response.content) / 1024
            print(f"  Saved: {filename} ({size_kb:.0f} KB)")
            return True
        elif response.status_code == 503:
            print(f"  Model loading, waiting 20s... (attempt {attempt + 1})")
            time.sleep(20)
        else:
            print(f"  Error {response.status_code}: {response.text}")
            return False
    return False


if __name__ == "__main__":
    out = "public/presentation-images"

    # Slide 2 "Why" background — abstract terminal/code aesthetic
    generate(
        "Minimal dark abstract banner, futuristic terminal aesthetic, "
        "deep navy and dark cyan gradients, subtle geometric grid lines, "
        "glowing cyan neon accents, circuit board traces fading into darkness, "
        "scanline texture overlay, vintage computer terminal feel, "
        "very dark background #0a0a0f, muted colors, no text, no people, "
        "cinematic wide composition, film grain",
        f"{out}/slide-02-why.png",
        width=1280,
        height=480,
    )

    # Slide 3 "Problem" — fragmented terminal illustration
    generate(
        "Abstract minimal dark illustration, cyberpunk style, "
        "a fragmented broken terminal screen with scattered code snippets, "
        "dark background #0a0a0f, vintage analog aesthetic, "
        "cyan and purple neon glow, subtle grid pattern, "
        "no text, no people, centered composition, film grain texture",
        f"{out}/slide-02-problem.svg",  # Will be PNG despite extension
        width=640,
        height=640,
    )

    # Slide 7 "Multi-Provider" — connected nodes/APIs
    generate(
        "Abstract minimal dark illustration, futuristic network style, "
        "multiple glowing nodes connected by light beams in a constellation, "
        "dark background #0a0a0f, cyan purple and green neon colors, "
        "each node represents a different AI provider, "
        "no text, no people, centered composition, clean lines",
        f"{out}/slide-06-providers.svg",  # Will be PNG despite extension
        width=640,
        height=640,
    )

    print("\nDone! Images saved to website/public/presentation-images/")
