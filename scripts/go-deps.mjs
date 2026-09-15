#!/usr/bin/env node
/**
 * Friendly dependency refresh (`go mod download` + `go mod tidy`).
 *
 * The commands are quiet; what you actually want to know is whether tidy
 * rewrote go.mod or go.sum, and by how much. That is the summary.
 *
 * Usage:
 *   node scripts/go-deps.mjs            # download + tidy + summary
 *   node scripts/go-deps.mjs --verbose
 *   node scripts/go-deps.mjs --show-log
 *
 * Env: see lib/friendly.mjs
 */

import path from 'node:path';
import {
  ROOT,
  bold,
  cli,
  dim,
  fail,
  green,
  header,
  logPath,
  red,
  rel,
  runStep,
  safeRead,
  showLog,
  yellow,
} from './lib/friendly.mjs';

const opts = cli();
const LOG = logPath('go-deps');

const snap = () => ({
  'go.mod': safeRead(path.join(ROOT, 'go.mod')) || '',
  'go.sum': safeRead(path.join(ROOT, 'go.sum')) || '',
});

main().catch((err) => {
  console.error(`${red('✗')} go-deps wrapper crashed: ${err?.stack || err}`);
  process.exit(1);
});

async function main() {
  if (opts.showLog) return showLog('go-deps');

  const before = snap();
  const started = Date.now();

  if (!opts.quiet) header('📦 Updating dependencies', '(go mod download + tidy)');

  const steps = [
    ['go mod download', ['mod', 'download']],
    ['go mod tidy', ['mod', 'tidy']],
  ];

  for (const [label, args] of steps) {
    const { code } = await runStep(label, 'go', args, {
      cwd: ROOT,
      logPath: LOG,
      append: label !== steps[0][0],
      verbose: opts.verbose,
    });
    if (code !== 0) {
      return fail(code, `${label} failed`, {
        logPath: LOG,
        hint: `   check go.mod and network access, then re-run`,
      });
    }
  }

  const after = snap();
  const changes = Object.keys(after).filter((f) => after[f] !== before[f]);

  console.log(`${green('✅')} ${bold('Dependencies up to date')}  ${dim(`${((Date.now() - started) / 1000).toFixed(1)}s`)}`);

  if (!changes.length) {
    console.log(`   ${dim('go.mod and go.sum unchanged')}`);
  } else {
    for (const file of changes) {
      const { added, removed } = lineDiff(before[file], after[file]);
      console.log(`   ${yellow('•')} ${bold(file)} ${dim(`+${added} −${removed}`)}`);
    }
    console.log(`   ${dim(`review with: git diff ${changes.join(' ')}`)}`);
  }

  console.log('');
  console.log(`   ${dim(`full log: ${rel(LOG)}`)}`);
  console.log('');
}

/** Line-level add/remove counts — enough to say "this file moved" without a real diff. */
function lineDiff(oldText, newText) {
  const count = (t) => {
    const m = new Map();
    for (const line of t.split('\n')) m.set(line, (m.get(line) || 0) + 1);
    return m;
  };
  const a = count(oldText);
  const b = count(newText);
  let added = 0;
  let removed = 0;
  for (const [line, n] of b) added += Math.max(0, n - (a.get(line) || 0));
  for (const [line, n] of a) removed += Math.max(0, n - (b.get(line) || 0));
  return { added, removed };
}
