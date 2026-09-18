# Horse build splash

`just build` clears an interactive terminal and shows a rotating, original low-poly
horse using [Rasterminal](https://github.com/PavolUlicny/rasterminal) while compiling.
It includes an animated WEE wordmark, a type-in tagline, live frontend/Go stage
labels, elapsed time and an indeterminate loading track. On macOS/Linux, Python 3
bridges the renderer through a PTY so branding is composed into each frame without
flicker. Windows retains the plain rotating-horse view.
It still only builds; launch the resulting application with `./wee`.

Install the optional renderer with `just setup-splash` (Git, CMake 3.22+ and a
C++17 compiler required). This builds pinned upstream revision
`7c4905c50fdd880709c1e8097cbc170970c22f6f` into ignored `dist/tools/`.
No downloads occur during normal builds. A Rasterminal binary on PATH or selected
with `WEE_RASTERMINAL` works too. Without it, builds use a simple horse heading.

`WEE_SPLASH=0 just build` disables the splash. CI, redirected input/output,
`TERM=dumb`, `NO_COLOR`, and `WEE_VERBOSE=1` skip it automatically.
`just build-verbose` retains the full streamed build output. Logs remain in
`dist/logs/`; failures retain their exit status and skip later steps.
Q dismisses the horse while the build continues; Ctrl+C cancels the build.
The renderer uses one thread at 12 FPS and the broadly supported block backend.

`horse.obj` and `horse.mtl` are original project assets. Regenerate the mesh with
`node scripts/assets/make-horse.mjs`. No third-party model or texture is bundled.
Rasterminal itself is MIT licensed; its dependencies have separate notices, kept
with its source by the setup script. No renderer binary is committed.
