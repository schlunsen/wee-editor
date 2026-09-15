#!/usr/bin/env node
/**
 * Friendly `go build` wrapper.
 *
 * Hides cgo/GCC chatter (sherpa-onnx ships a deprecated C API, so every clean
 * build prints a deprecation block) and reports the one thing you actually want
 * to know: how big the binary got.
 *
 * Usage:
 *   node scripts/build-go.mjs              # build + summary
 *   node scripts/build-go.mjs --verbose    # stream the complete output
 *   node scripts/build-go.mjs --show-log   # print the last build log
 *   node scripts/build-go.mjs --no-warnings
 *   node scripts/build-go.mjs -o ./other   # build to a different path
 *
 * Env: see lib/friendly.mjs
 */

import path from 'node:path';
import {
  ROOT,
  bold,
  cli,
  dim,
  fileSize,
  formatBytes,
  formatDuration,
  header,
  logPath,
  printWarnings,
  red,
  rel,
  runStep,
  fail,
  green,
  showLog,
  truncate,
} from './lib/friendly.mjs';

const opts = cli();
const LOG = logPath('go-build');

const WEE = process.platform === 'win32' ? 'wee.exe' : 'wee';
const OUT = opts.value('-o') || opts.value('--output') || WEE;
const OUT_PATH = path.resolve(ROOT, OUT);

main().catch((err) => {
  console.error(`${red('✗')} build-go wrapper crashed: ${err?.stack || err}`);
  process.exit(1);
});

async function main() {
  if (opts.showLog) return showLog('go-build');

  const before = fileSize(OUT_PATH);

  if (!opts.quiet) header('🔨 Building wee', `(Go → ./${OUT})`);

  const warnings = [];
  const errors = [];
  let current = null;

  const { code, elapsed } = await runStep(`go build -o ${OUT} ./cmd/wee`, 'go', [
    'build',
    '-o',
    OUT,
    './cmd/wee',
  ], {
    cwd: ROOT,
    logPath: LOG,
    verbose: opts.verbose,
    // cgo is opt-in on Windows; sqlite needs it.
    env: process.platform === 'win32' ? { CGO_ENABLED: '1' } : {},
    onLine: (line) => {
      // `# <package>` starts a new compiler-output block; `warning:`/`note:`
      // lines belong to whichever block is open.
      if (line.startsWith('# ')) {
        current = line.slice(2).trim();
        return;
      }
      const warn = line.match(/warning:\s*(.*)$/);
      if (warn) {
        warnings.push({ pkg: current, text: warn[1].trim() });
        return;
      }
      // Go's own diagnostics: `./file.go:12:3: undefined: thing`.
      if (/^\.?\.?\/?[\w./-]+\.go:\d+:\d+:\s/.test(line) || /^(undefined|too many|cannot use)/.test(line)) {
        errors.push(line.trim());
      }
    },
  });

  // Compiler errors are the whole point of a failed build — show them, don't
  // bury them in a log tail.
  if (code !== 0) {
    for (const e of errors.slice(0, 40)) console.error(`   ${red('•')} ${e}`);
    return fail(code, 'Go build failed', {
      logPath: LOG,
      hint: `   re-run with full output: ${dim('just build-verbose')}`,
    });
  }

  const after = fileSize(OUT_PATH);
  const delta = before ? after - before : 0;
  const deltaText = !before
    ? ''
    : delta === 0
      ? dim('unchanged')
      : delta > 0
        ? `+${formatBytes(delta)}`
        : `−${formatBytes(Math.abs(delta))}`;

  console.log(
    `${green('✅')} ${bold(`Built ./${OUT}`)}  ${dim(
      `(${formatBytes(after)}${deltaText ? `, ${deltaText}` : ''})  in ${formatDuration(elapsed)}`,
    )}`,
  );
  console.log('');

  const { actionable, benign } = digest(warnings);
  printWarnings({ actionable, benign, log: LOG, show: !opts.noWarnings });

  if (opts.verbose) console.log(dim(`   full log: ${rel(LOG)}`));
  console.log('');
}

/**
 * Split compiler warnings into "something you can act on in this repo" and
 * "a dependency has a deprecated C API", which is what the sherpa-onnx cgo
 * preamble produces on every clean build.
 */
function digest(warnings) {
  const actionable = [];
  const benign = [];
  const seen = new Set();

  for (const w of warnings) {
    const isDeprecation = /deprecated/i.test(w.text);
    const isDependency = w.pkg && !w.pkg.startsWith('github.com/schlunsen/wee-editor');

    if (isDeprecation && isDependency) {
      const label = `${shortPkg(w.pkg)} — deprecated C API (cgo)`;
      if (!seen.has(label)) {
        seen.add(label);
        benign.push(label);
      }
      continue;
    }

    const text = truncate(`${w.pkg ? `${shortPkg(w.pkg)}: ` : ''}${w.text}`, 150);
    if (!seen.has(text)) {
      seen.add(text);
      actionable.push(text);
    }
  }

  return { actionable, benign };
}

/** `github.com/k2-fsa/sherpa-onnx-go-macos` → `sherpa-onnx-go-macos`. */
const shortPkg = (p) => p.split('/').pop();
