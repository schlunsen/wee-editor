#!/usr/bin/env node
// A build-only splash. The server is never started or stopped by this wrapper.
import fs from 'node:fs';
import path from 'node:path';
import { spawn, spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { StringDecoder } from 'node:string_decoder';
import { splashOverlay, frameCompositor } from './lib/splash-ui.mjs';
import { ROOT } from './lib/friendly.mjs';

export function splashEnabled({ stdout = process.stdout, stdin = process.stdin, env = process.env } = {}) {
  return Boolean(stdout.isTTY && stdin.isTTY && env.TERM !== 'dumb' && !env.CI &&
    env.NO_COLOR === undefined && env.WEE_SPLASH !== '0' && env.WEE_VERBOSE !== '1');
}

export async function build({ steps, renderer, splash = splashEnabled() } = {}) {
  steps ||= ['build-frontend.mjs', 'build-go.mjs'].map(file => [process.execPath, [path.join(ROOT, 'scripts', file)]]);
  const local = path.join(ROOT, 'dist', 'tools', process.platform === 'win32' ? 'rasterminal.exe' : 'rasterminal');
  renderer ||= process.env.WEE_RASTERMINAL || (fs.existsSync(local) ? local : 'rasterminal');
  let viewer, child, stoppingViewer = false, interrupted = 0;
  let buffered = '', buffering = false;
  let viewerDone = Promise.resolve();
  const started = Date.now();
  let stage = 0;
  const flush = () => { buffering = false; if (buffered) process.stdout.write(buffered); buffered = ''; };
  // Bound memory even if a verbose tool unexpectedly prints a great deal.
  const output = data => { if (buffering) buffered = (buffered + data).slice(-1024*1024); else process.stdout.write(data); };
  const stopBuild = signal => {
    interrupted = signal === 'SIGINT' ? 130 : 143;
    if (!child?.pid) return;
    try {
      if (process.platform === 'win32') spawnSync('taskkill', ['/pid', String(child.pid), '/T', '/F'], { stdio: 'ignore' });
      else process.kill(-child.pid, 'SIGTERM');
    } catch { /* process already finished */ }
  };
  const onInt = () => stopBuild('SIGINT');
  const onTerm = () => stopBuild('SIGTERM');
  process.on('SIGINT', onInt);
  process.on('SIGTERM', onTerm);
  let savedTerminal;
  if (splash && process.platform !== 'win32') {
    savedTerminal = spawnSync('stty', ['-g'], { stdio: ['inherit', 'pipe', 'ignore'], encoding: 'utf8' }).stdout?.trim();
  }
  try {
    if (splash) {
      process.stdout.write('\x1b[2J\x1b[H');
      buffering = true;
      const rendererArgs = ['--graphics', 'blocks', '--threads', '1', '--fps', '12', '--spin',
        '--spin-speed', '18', '--no-input', '--no-hud', '--no-ao', '--shading', 'flat',
        '--zoom', '0.60', '--yaw', '25', path.join(ROOT, 'scripts', 'assets', 'horse.obj')];
      const composed = process.platform !== 'win32';
      viewer = composed
        ? spawn('python3', [path.join(ROOT, 'scripts', 'lib', 'splash-terminal.py'), renderer, ...rendererArgs], { stdio: ['inherit', 'pipe', 'ignore'] })
        : spawn(renderer, rendererArgs, { stdio: ['inherit', 'inherit', 'ignore'] });
      const compositor = frameCompositor(() => splashOverlay({
        columns: process.stdout.columns, rows: process.stdout.rows,
        elapsed: Date.now() - started, stage, total: steps.length
      }), text => process.stdout.write(text));
      const decoder = new StringDecoder('utf8');
      viewer.stdout?.on('data', data => compositor.feed(decoder.write(data)));
      viewerDone = new Promise(resolve => {
        viewer.once('error', () => {
          output('\n  🐎 Wee — building\n  Optional 3D splash: just setup-splash\n\n');
        });
        viewer.once('close', (code, signal) => {
          if (composed) { compositor.feed(decoder.end()); compositor.flush(); }
          if (code === 127) output('\n  wee — building\n  Optional 3D splash: just setup-splash\n\n');
          if (!stoppingViewer && (signal === 'SIGINT' || code === 130)) stopBuild('SIGINT');
          if (savedTerminal) spawnSync('stty', [savedTerminal], { stdio: ['inherit', 'ignore', 'ignore'] });
          process.stdout.write('\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?1049l\x1b[?25h');
          flush();
          resolve();
        });
      });
    }
    for (const [index, [command, args]] of steps.entries()) {
      stage = index;
      if (interrupted) break;
      const code = await new Promise(resolve => {
        child = spawn(command, args, { cwd: ROOT, detached: process.platform !== 'win32', stdio: ['ignore', 'pipe', 'pipe'] });
        child.stdout.on('data', output);
        child.stderr.on('data', output);
        child.once('error', error => output(`${error.message}\n`));
        child.once('close', code => resolve(code ?? 1));
      });
      child = null;
      if (code !== 0 || interrupted) return interrupted || code;
    }
    if (interrupted) return interrupted;
    output(`\n🎉 ${process.platform === 'win32' ? 'wee.exe' : 'wee'} is ready. Run ./${process.platform === 'win32' ? 'wee.exe' : 'wee'}\n`);
    return 0;
  } finally {
    stoppingViewer = true;
    if (viewer?.pid && viewer.exitCode === null && viewer.signalCode === null) {
      viewer.kill('SIGTERM');
      const timeout = setTimeout(() => viewer.kill('SIGKILL'), 2000);
      await viewerDone;
      clearTimeout(timeout);
    } else await viewerDone;
    if (savedTerminal) spawnSync('stty', [savedTerminal], { stdio: ['inherit', 'ignore', 'ignore'] });
    flush();
    process.off('SIGINT', onInt);
    process.off('SIGTERM', onTerm);
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  build().then(code => { process.exitCode = code; }).catch(error => { console.error(error); process.exitCode = 1; });
}
