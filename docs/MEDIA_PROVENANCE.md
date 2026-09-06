# Media provenance

The root `LICENSE` (MIT) applies to Wee's source code. It is **not** asserted over
third-party media. This file records where the images and audio tracked in this
repository came from, so that anyone redistributing the repository can check the
terms that actually apply.

Last reviewed: 2026-09-06.

## Generated in this repository

| Asset group | Files | How it was produced |
| --- | --- | --- |
| Cat avatars | `internal/server/frontend/public/avatars/cats/*.jpg` (74) | Text-to-image, `black-forest-labs/FLUX.1-schnell` via the Hugging Face Inference API. The prompts and generator are checked in: `scripts/generate_cat_avatars.sh`, `scripts/generate_cat_avatars_dali.py`, `scripts/generate_cat_avatars_line*.py`. |
| Website feature icons | `website/public/images/icon-*.png` | Text-to-image, FLUX.1 (commit `d9b99212`). |
| Website illustrations | `website/public/images/hero-illustration.png`, `feature-*.png`, `logo-cat-*.png` | Text-to-image, FLUX.1 (commit `d9b99212`). |
| Hacker-cat logo | `website/public/images/logo-hacker-cat.png`, `ios/.../HackerCat.imageset/hacker-cat.png`, `src-tauri/ui/logo.png` | Text-to-image, FLUX.1 (commit `230c4a92`). |
| Presentation slide graphics | `website/public/presentation-images/*.svg` | Authored SVG; generator for the raster variants is `website/generate-graphics.py` (FLUX.1). |
| Presentation narration | `website/public/presentation-audio/slide-01..10.mp3` | Text-to-speech, Kokoro TTS, `af_sky` voice (commit `bfec488e`). Synthetic speech; no third-party recording or transcript. |
| Diagrams | `docs/architecture.svg`, `docs/architecture.png`, `docs/loop-controller-flow.svg`, `docs/loop-controller-flow.png` | Hand-authored SVG (plus PNG renders) created for this project. |
| App icons / favicons | `internal/server/frontend/public/favicon.svg`, `favicon.ico`, `pwa-*.png`, `website/public/favicon.ico`, `src-tauri/icons/**`, `ios/.../AppIcon.appiconset/**` | Derived from the project's own cat logo artwork. |
| Banner | `docs/banner.png` | Project artwork (commit `0b5999b2`). |

Generative-model output has no settled, universal copyright status. These entries
record how each asset was made; they are not a legal opinion, and they do not
substitute for reviewing the terms of the model and API used.

## Screenshots

`docs/*.png` and `website/public/images/*.png` screenshots are captures of Wee's
own UI. Screenshots that showed real session content, absolute personal paths,
third-party character artwork or a named remote server have been removed. Any new
screenshot must be taken against demo data on a local host before it is committed.

## Vendored third-party code with embedded notices

| File | Origin | License |
| --- | --- | --- |
| `website/public/three.min.js` | three.js | MIT (notice retained in the file header) |

## Removed for lack of redistribution permission

| File | Reason | Removed in |
| --- | --- | --- |
| `website/public/presentation-audio/bg-music.mp3` | Metadata identifies "Born to Win" by Aylex (publisher: Free To Use Music). The provider's [license](https://freetouse.com/license) permits attributed use in end-user content but separately prohibits making the digital asset available to third parties; redistributing the raw MP3 in a public repository is not supported by the permissions available at review time. | this cleanup |
| `src-tauri/ui/roar.mp3` | Lion-roar sound effect with encoder metadata only. No source or license recorded in its commit (`bd3976ab`), and none was supplied. | this cleanup |

Neither track is required for the software to build or run. If licensed
replacements are wanted later, add them together with a row in this table naming
the source and the terms.

## Bundled binaries

See [`../licenses/THIRD_PARTY_NOTICES.md`](../licenses/THIRD_PARTY_NOTICES.md)
for the ONNX Runtime and sherpa-onnx libraries shipped in macOS release archives.

## Still open

- Authority to publish contributions made on behalf of an employer or client is
  not established by scanning Git history and remains for the repository owner to
  confirm.
