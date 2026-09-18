import test from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { splashEnabled } from '../build.mjs';

test('splash stays out of CI, pipes, plain terminals and opt-out builds', () => {
  const tty = { isTTY: true };
  const allowed = { stdout: tty, stdin: tty, env: { TERM: 'xterm' } };
  assert.equal(splashEnabled(allowed), true);
  for (const env of [{ CI: '1' }, { NO_COLOR: '' }, { WEE_SPLASH: '0' }, { TERM: 'dumb' }, { WEE_VERBOSE: '1' }]) {
    assert.equal(splashEnabled({ ...allowed, env }), false);
  }
  assert.equal(splashEnabled({ ...allowed, stdout: { isTTY: false } }), false);
  assert.equal(splashEnabled({ ...allowed, stdin: { isTTY: false } }), false);
});

test('a failed build preserves output and status and never runs the next step', () => {
  const script = `import { build } from './scripts/build.mjs';
    process.exitCode = await build({ splash: false, steps: [
      [process.execPath, ['-e', 'console.error("expected failure"); process.exit(7)']],
      [process.execPath, ['-e', 'console.log("must not run")']]
    ] });`;
  const result = spawnSync(process.execPath, ['--input-type=module', '-e', script], { encoding: 'utf8' });
  assert.equal(result.status, 7);
  assert.match(result.stdout, /expected failure/);
  assert.doesNotMatch(result.stdout, /must not run|ready|\x1b/);
});

test('missing renderer falls back and does not prevent a successful build', () => {
  const script = `import { build } from './scripts/build.mjs';
    process.exitCode = await build({ splash: true, renderer: '/missing/rasterminal',
      steps: [[process.execPath, ['-e', 'console.log("build completed")']]] });`;
  const result = spawnSync(process.execPath, ['--input-type=module', '-e', script], { encoding: 'utf8' });
  assert.equal(result.status, 0);
  assert.match(result.stdout, /build completed/);
  assert.match(result.stdout, /ready/);
});

test('branding and build-stage text are inserted atomically, even with split frames', async () => {
  const { splashOverlay, frameCompositor } = await import('../lib/splash-ui.mjs');
  const chunks = [];
  const compositor = frameCompositor(() => 'BRANDING', s => chunks.push(s));
  compositor.feed('\x1b[?2026hHORSE\x1b[?20');
  assert.equal(chunks.length, 0);
  compositor.feed('26l');
  assert.equal(chunks[0], '\x1b[?2026hHORSEBRANDING\x1b[?2026l');
  compositor.feed('\x1b[?2026l');
  assert.equal(chunks[1], '\x1b[?2026l');
  const frame = splashOverlay({ columns: 100, rows: 36, elapsed: 5000, stage: 1 });
  assert.match(frame, /Your agents\. One place\./);
  assert.match(frame, /Compiling the engine/);
  assert.match(frame, /Frontend ready/);
  assert.match(frame, /0:05/);
});
