#!/usr/bin/env node
/**
 * Friendly cross-platform build.
 *
 * Compiles the five release targets one at a time and prints a table — target,
 * size, or the reason it could not be built. `go build` on its own just dumps
 * one "build constraints exclude all Go files" trace per failure, which tells
 * you nothing about the cause.
 *
 * That cause is cgo: sqlite (mattn/go-sqlite3) and sherpa-onnx both need it, and
 * Go disables cgo whenever GOOS/GOARCH differ from the host. So from a Mac you
 * get darwin/arm64 and nothing else; the other four need a cross toolchain or a
 * native runner. CI does exactly that — see the `release` recipe.
 *
 * Usage:
 *   node scripts/build-all.mjs              # build every target, summarise
 *   node scripts/build-all.mjs --verbose    # stream the complete output
 *   node scripts/build-all.mjs --show-log   # print the last build log
 *   node scripts/build-all.mjs darwin/arm64 linux/amd64   # only these targets
 *
 * Env: see lib/friendly.mjs
 */

import fs from 'node:fs';
import path from 'node:path';
import {
  ROOT,
  bold,
  cli,
  cyan,
  dim,
  fileSize,
  formatBytes,
  green,
  header,
  logPath,
  red,
  rel,
  runStep,
  showLog,
  truncate,
  yellow,
} from './lib/friendly.mjs';

const ALL_TARGETS = [
  'linux/amd64',
  'linux/arm64',
  'darwin/amd64',
  'darwin/arm64',
  'windows/amd64',
];

const opts = cli();
const LOG = logPath('build-all');
const DIST = path.join(ROOT, 'dist');

// Positional args narrow the matrix: `just build-all darwin/arm64`.
const requested = opts.argv.filter((a) => /^[\w-]+\/[\w-]+$/.test(a));
const targets = requested.length ? requested : ALL_TARGETS;

main().catch((err) => {
  console.error(`${red('✗')} build-all wrapper crashed: ${err?.stack || err}`);
  process.exit(1);
});

async function main() {
  if (opts.showLog) return showLog('build-all');

  if (requested.length) {
    const unknown = requested.filter((t) => !ALL_TARGETS.includes(t));
    if (unknown.length) {
      console.error(`${red('✗')} unknown target${unknown.length === 1 ? '' : 's'}: ${unknown.join(', ')}`);
      console.error(`   known targets: ${ALL_TARGETS.join(', ')}`);
      process.exit(2);
    }
  }

  if (!opts.quiet) {
    const what = requested.length ? 'Building selected platforms' : 'Building for all platforms';
    header(`📦 ${what}`, `(${targets.length} target${targets.length === 1 ? '' : 's'} → dist/)`);
  }

  fs.mkdirSync(DIST, { recursive: true });
  fs.writeFileSync(LOG, ''); // fresh log; each target appends its own section

  const results = [];
  for (const target of targets) {
    results.push(await build(target));
  }

  printTable(results);

  const failed = results.filter((r) => !r.ok);
  if (failed.length) process.exit(1);
}

async function build(target) {
  const [goos, goarch] = target.split('/');
  const name = `wee-${goos}-${goarch}${goos === 'windows' ? '.exe' : ''}`;
  const outFile = path.join(DIST, name);

  const { code, elapsed } = await runStep(`go build ${target}`, 'go', ['build', '-o', path.join('dist', name), './cmd/wee'], {
    cwd: ROOT,
    logPath: LOG,
    append: true,
    verbose: opts.verbose,
    env: { GOOS: goos, GOARCH: goarch },
  });

  if (code !== 0) {
    return { target, ok: false, reason: reasonFor(failuresIn(fs.readFileSync(LOG, 'utf8').split(/\r?\n──── /).pop() || '')) };
  }

  // A darwin/arm64 binary built on an arm Mac is meaningless on linux; only
  // report sizes for the targets that actually produced something.
  return { target, ok: true, size: fileSize(outFile), elapsed, name };
}

/**
 * Reduce a Go cross-compile failure to its cause. The overwhelmingly common one
 * here is cgo being switched off for a foreign platform, which shows up as
 * "build constraints exclude all Go files in …/sherpa-onnx-go-linux".
 */
function failuresIn(text) {
  return text
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith('#') && !l.startsWith('→'));
}

function reasonFor(lines) {
  const cgo = lines.find((l) => /build constraints exclude all Go files in .*sherpa-onnx-go-/.test(l));
  if (cgo) {
    const pkg = cgo.match(/sherpa-onnx-go-([\w-]+)/)?.[1] ?? 'cgo deps';
    return `${pkg} needs cgo — disabled when cross-compiling`;
  }
  if (lines.some((l) => /CGO_ENABLED=0/.test(l))) return 'cgo is disabled for this platform';

  const first = lines.find((l) => /\berror\b|cannot|undefined/i.test(l)) || lines[0] || 'unknown error';
  return truncate(first.replace(/^[\w./@-]+:\s*/, ''), 90);
}

function printTable(results) {
  const width = Math.max(...results.map((r) => r.target.length));

  for (const r of results) {
    const pad = r.target.padEnd(width);
    if (r.ok) {
      console.log(`   ${green('✓')} ${pad}  ${dim(formatBytes(r.size).padStart(9))}`);
    } else {
      console.log(`   ${red('✗')} ${pad}  ${dim(r.reason)}`);
    }
  }
  console.log('');

  const ok = results.filter((r) => r.ok);
  const bad = results.filter((r) => !r.ok);

  if (ok.length) {
    console.log(
      `${green('✅')} ${bold(`Built ${ok.length} of ${results.length}`)}  ${dim(
        `→ dist/  (${formatBytes(ok.reduce((s, r) => s + r.size, 0))} total)`,
      )}`,
    );
  } else {
    console.log(`${red('✗')} ${bold(`Built 0 of ${results.length}`)}`);
  }

  if (bad.length) {
    console.log('');
    console.log(
      `   ${yellow('⚠')}  ${bad.length} target${bad.length === 1 ? '' : 's'} need a cross-compiling toolchain (cgo).`,
    );
    console.log(`      ${dim('CI builds them on native runners — push a tag with:')} ${cyan('just release <version> <name>')}`);
  }

  console.log('');
  console.log(`   ${dim(`full log: ${rel(LOG)}`)}`);
  console.log('');
}
