#!/usr/bin/env python3
"""Generate presentation narration audio using Kokoro TTS (local).

Uses the af_sky voice — a natural, young woman's voice.

Usage:
  python website/generate-narration.py
  python website/generate-narration.py --slide 3
"""

import argparse
import io
import os
import subprocess
import shutil
import sys

import numpy as np
import soundfile as sf

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
OUTPUT_DIR = os.path.join(SCRIPT_DIR, "public", "presentation-audio")

VOICE = "af_sky"       # Young woman's voice
SPEED = 1.0
LANG = "a"             # American English

# ---------------------------------------------------------------------------
# Narration scripts for each slide
# ---------------------------------------------------------------------------

SLIDE_NARRATIONS = {
    1: (
        "Wee. The control center for Claude Code. "
        "Run, monitor, and control your AI agents — all from a single fifteen megabyte binary. "
        "Live sessions. Real-time analytics. Multi-provider support. "
        "Let's take a look."
    ),
    2: (
        "AI coding is exploding. Developers run dozens of Claude sessions every day, "
        "but without a control plane, you lose context, burn tokens, and have zero visibility "
        "into what your agents are doing. "
        "The terminal isn't enough. CLI sessions disappear when you close the tab. "
        "No history, no metrics, no way to run agents side-by-side. "
        "Wee is the missing layer — a self-hosted control center that wraps Claude Code "
        "with live sessions, real-time analytics, voice input, and multi-provider support."
    ),
    3: (
        "The problem with AI agents today? "
        "Running Claude Code in a terminal is powerful, but managing multiple sessions, "
        "tracking costs, and maintaining context across projects is a nightmare. "
        "Zero visibility into token spend. Lost context on every restart. "
        "And only one session at a time in the CLI."
    ),
    4: (
        "Meet Wee. A complete control plane for your AI coding workflow. "
        "At its core, Wee connects live sessions, real-time analytics, "
        "multi-provider support, and granular permissions — all orchestrated "
        "through a single hub."
    ),
    5: (
        "How does Wee work? Four pillars. One seamless experience. "
        "First, connect — start a live agent session via your browser. "
        "WebSocket streams every tool call in real-time. "
        "Second, control — granular permissions, YOLO mode toggle, "
        "and provider selection, all from the UI. "
        "Third, monitor — real-time token usage, costs, tool executions, "
        "and session metrics on the dashboard. "
        "And fourth, persist — SQLite storage keeps sessions, handoff context, "
        "and analytics across restarts."
    ),
    6: (
        "Live agent sessions. Run multiple Claude sessions side-by-side in your browser. "
        "Every tool call — Bash, Read, Write, Edit — streams live via WebSocket. "
        "Prompt, execute, stream, render. All in real-time."
    ),
    7: (
        "Multi-provider support. Switch between Claude, DeepSeek, GLM, Kimi, "
        "Z.AI, or any Anthropic-compatible API — without changing a line of code. "
        "One interface, six-plus providers."
    ),
    8: (
        "The real-time dashboard gives you full visibility. "
        "Token usage. Costs. Tool calls. Git status. All live. "
        "See every active session, which provider it's using, "
        "and exactly how much it's costing you."
    ),
    9: (
        "Built for speed. A single fifteen megabyte Go binary. "
        "No Node.js, no Docker required. "
        "Fifty to one hundred times faster than JavaScript alternatives. "
        "Go for zero-dependency deployment. SQLite for persistent sessions. "
        "WebSocket for real-time streaming. "
        "Built-in voice input with Parakeet speech-to-text. "
        "And optional Docker for single-command deploy."
    ),
    10: (
        "Take control of your AI coding workflow. "
        "Live sessions. Multi-provider. Real-time analytics. "
        "Install Wee with a single command and get started today. "
        "Check it out on GitHub."
    ),
}

# ---------------------------------------------------------------------------
# Kokoro TTS
# ---------------------------------------------------------------------------

_pipeline = None

def get_pipeline():
    global _pipeline
    if _pipeline is None:
        from kokoro import KPipeline
        print("Initializing Kokoro TTS pipeline...")
        _pipeline = KPipeline(lang_code=LANG)
    return _pipeline


def generate_slide_audio(slide_num: int, text: str):
    """Generate narration for a single slide and save as MP3."""
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    padded = str(slide_num).zfill(2)
    wav_path = os.path.join(OUTPUT_DIR, f"slide-{padded}.wav")
    mp3_path = os.path.join(OUTPUT_DIR, f"slide-{padded}.mp3")

    print(f"\n--- Slide {slide_num} ---")
    print(f"  Text: {text[:80]}...")

    pipeline = get_pipeline()

    # Generate audio chunks
    all_audio = []
    for gs, ps, audio in pipeline(text, voice=VOICE, speed=SPEED):
        all_audio.append(audio)

    if not all_audio:
        print(f"  ERROR: No audio generated for slide {slide_num}")
        return False

    # Concatenate chunks
    combined = np.concatenate(all_audio) if len(all_audio) > 1 else all_audio[0]

    # Save as WAV
    sf.write(wav_path, combined, 24000, format="WAV")
    print(f"  WAV saved: {wav_path}")

    # Convert to MP3 if ffmpeg available
    if shutil.which("ffmpeg"):
        subprocess.run(
            ["ffmpeg", "-y", "-i", wav_path, "-codec:a", "libmp3lame", "-qscale:a", "2", mp3_path],
            check=True, capture_output=True,
        )
        os.remove(wav_path)
        size_kb = os.path.getsize(mp3_path) / 1024
        print(f"  MP3 saved: {mp3_path} ({size_kb:.0f} KB)")
    else:
        print("  WARNING: ffmpeg not found, keeping WAV format")
        # Rename to mp3 path so the presentation can find it (won't be valid mp3)
        os.rename(wav_path, mp3_path)

    return True


def main():
    parser = argparse.ArgumentParser(description="Generate Wee presentation narration")
    parser.add_argument("--slide", type=int, help="Generate only this slide number (1-10)")
    args = parser.parse_args()

    if args.slide:
        if args.slide not in SLIDE_NARRATIONS:
            print(f"Error: slide {args.slide} not found (valid: 1-10)")
            sys.exit(1)
        generate_slide_audio(args.slide, SLIDE_NARRATIONS[args.slide])
    else:
        for slide_num, text in SLIDE_NARRATIONS.items():
            generate_slide_audio(slide_num, text)

    print(f"\nDone! Audio files saved to {OUTPUT_DIR}/")


if __name__ == "__main__":
    main()
