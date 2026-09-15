#!/usr/bin/env node
/**
 * Friendly Nuxt frontend build wrapper.
 *
 * Runs `npm run generate` in internal/server/frontend and hides the ~460 lines
 * of Nuxt/Vite/Nitro chatter behind a spinner plus a compact summary. On failure
 * it shows the tail of the log and points at the full log file.
 *
 * Usage:
 *   node scripts/build-frontend.mjs              # quiet build + summary
 *   node scripts/build-frontend.mjs --verbose    # stream everything live
 *   node scripts/build-frontend.mjs --show-log   # print the last build log
 *   node scripts/build-frontend.mjs --no-warnings
 *   node scripts/build-frontend.mjs --install    # dependencies only
 *
 * Env:
 *   WEE_BUILD_VERBOSE=1   same as --verbose
 *   WEE_BUILD_LOG=<path>  override log location
 *   NO_COLOR=1            disable colour
 *
 * Shared plumbing lives in lib/friendly.mjs.
 */

import fs from 'node:fs';
import path from 'node:path';
import {
  FRONTEND_DIR,
  bold,
  cli,
  dim,
  fail,
  formatBytes,
  green,
  header,
  logPath,
  printWarnings,
  red,
  rel,
  relativize,
  runStep,
  safeRead,
  showLog,
  stripAnsi,
  truncate,
} from './lib/friendly.mjs';

const opts = cli();
const OUTPUT_DIR = path.join(FRONTEND_DIR, '.output', 'public');
const LOG = process.env.WEE_BUILD_LOG || logPath('frontend-build');

main().catch((err) => {
  console.error(`${red('✗')} build wrapper crashed: ${err?.stack || err}`);
  process.exit(1);
});

async function main() {
  if (opts.showLog) return showLog(process.env.WEE_BUILD_LOG || 'frontend-build');

  const started = Date.now();
  const installOnly = opts.has('--install', '--deps');
  if (!opts.quiet && !installOnly) header('🎨 Building frontend', '(Nuxt → static)');

  if (!fs.existsSync(FRONTEND_DIR)) {
    console.error(`${red('✗')} Frontend directory not found: ${rel(FRONTEND_DIR)}`);
    console.error(`   This script must live in <repo>/scripts/ — expected ${FRONTEND_DIR}`);
    process.exit(1);
  }

  // 1. Dependencies (only when missing, or always with --install).
  if (installOnly) {
    await installDeps(logPath('frontend-install'));
    process.exit(0);
  }
  if (!fs.existsSync(path.join(FRONTEND_DIR, 'node_modules'))) {
    await installDeps();
  }

  // 2. The actual build.
  const { code } = await runStep('🎨 Generating static site', 'npm', ['run', 'generate'], {
    cwd: FRONTEND_DIR,
    logPath: LOG,
    verbose: opts.verbose,
  });
  if (code !== 0) return fail(code, 'Frontend build failed', { logPath: LOG, hint: `   stream it live next time: ${dim('just build-frontend-verbose')}` });

  // 3. Summary.
  const elapsed = ((Date.now() - started) / 1000).toFixed(1);
  const stats = outputStats(OUTPUT_DIR);
  const routes = readRouteCount();

  console.log(`${green('✅')} ${bold('Frontend built')} in ${bold(`${elapsed}s`)}`);
  const parts = [];
  if (routes) parts.push(`${routes} route${routes === 1 ? '' : 's'}`);
  if (stats.files) parts.push(`${stats.files.toLocaleString('en-US')} files`);
  if (stats.bytes) parts.push(formatBytes(stats.bytes));
  if (parts.length) console.log(`   ${dim(`${parts.join(' · ')} → .output/public`)}`);
  console.log('');

  const { actionable, benign } = extractWarnings(stripAnsi(safeRead(LOG) || ''));
  printWarnings({
    actionable,
    benign,
    log: opts.verbose ? null : LOG,
    show: !opts.noWarnings,
  });
  if (opts.verbose) console.log(dim(`   full log: ${rel(LOG)}`));
  console.log('');
}

// ── dependencies ─────────────────────────────────────────────────────────────

/**
 * Run `npm install` and report what it did — npm buries "added 1,234 packages
 * in 42s" (or "up to date") among its funding notices and audit summary.
 * Exits the process on failure; returns true otherwise.
 */
async function installDeps(log = LOG) {
  let summary = null;
  let noisy = 0;

  const { code, elapsed } = await runStep('📦 Installing frontend dependencies', 'npm', ['install'], {
    cwd: FRONTEND_DIR,
    logPath: log,
    verbose: opts.verbose,
    onLine: (line) => {
      const t = line.trim();
      if (/^(added|removed|changed|up to date)/i.test(t)) summary = t;
      else if (/packages? are looking for funding|vulnerabilit|npm notice/i.test(t)) noisy++;
    },
  });

  if (code !== 0) {
    fail(code, 'Dependency installation failed', { logPath: log });
    return false;
  }

  // The build path installs silently; only --install prints its own summary.
  if (opts.has('--install', '--deps')) {
    console.log(`\n${green('✅')} ${bold('Dependencies installed')}`);
    console.log(`   ${dim(summary || `in ${(elapsed / 1000).toFixed(1)}s`)}`);
    if (noisy) {
      console.log(`   ${dim(`ℹ  ${noisy} funding/audit notices suppressed — see the log`)}`);
    }
    console.log('');
    console.log(`   ${dim(`full log: ${rel(log)}`)}`);
    console.log('');
  }

  return true;
}

// ── warning digest ───────────────────────────────────────────────────────────

// Known-benign chatter that appears on every build and is not actionable inside
// this repo. Counted and named, never silently dropped.
const BENIGN = [
  { re: /baseline-browser-mapping/i, label: 'baseline-browser-mapping stale' },
  { re: /Browserslist|caniuse-lite/i, label: 'browserslist data stale' },
  { re: /Some chunks are larger than|chunkSizeWarningLimit/i, label: 'chunks > 500 kB' },
  { re: /HTML content not prerendered|ssr:\s*false/i, label: 'HTML not prerendered (ssr:false)' },
];

const MARKER_RE = /^\s*(?:\[[a-z]+\]\s*)?(?:WARN|⚠️?|ERROR|Error)\s*(.*)$/;
const STOP_RE = /^\s*(?:\[[a-z]+\]\s*)?(?:ℹ|✔|✓|WARN|⚠|ERROR|Error)/;

function extractWarnings(text) {
  const actionable = [];
  const benign = [];
  const seenActionable = new Set();
  const seenBenign = new Set();
  const lines = text.split(/\r?\n/);

  for (let i = 0; i < lines.length; i++) {
    const m = lines[i].match(MARKER_RE);
    if (!m) continue;

    // A warning's text may sit on the marker line, on the lines after it (Nuxt
    // emits a bare "WARN" then the body), or both. Gather it all into one blob.
    const parts = [m[1].trim()];
    for (let j = i + 1; j < lines.length && parts.length < 6; j++) {
      const next = lines[j];
      if (next.trim() === '' || STOP_RE.test(next)) break;
      parts.push(next.trim());
      i = j;
    }

    const blob = parts.filter(Boolean).join(' ');
    if (!blob || /^\[plugin vite:reporter\]$/i.test(blob)) continue;

    const benignMatch = BENIGN.find((b) => b.re.test(blob));
    if (benignMatch) {
      if (!seenBenign.has(benignMatch.label)) {
        seenBenign.add(benignMatch.label);
        benign.push(benignMatch.label);
      }
      continue;
    }

    const norm = normalize(blob);
    if (!norm || seenActionable.has(norm)) continue;
    seenActionable.add(norm);
    actionable.push(norm);
  }

  return { actionable, benign };
}

/** Collapse a raw log line into a stable, dedupe-friendly, path-free message. */
function normalize(msg) {
  const s = relativize(
    msg
      .replace(/\s+/g, ' ')
      .replace(/^[\s\-–•]+/, '')
      .replace(/^\(!\)\s*/, '')
      .trim(),
  );

  // Vite re-reports Nuxt's duplicate-import warnings in a slightly different
  // shape; rewriting both to one form lets them dedupe into a single entry.
  const dup = s.match(
    /^Duplicated imports "([^"]+)", the one from "([^"]+)" has been ignored and "([^"]+)" is used$/,
  );
  if (dup) return `Duplicated import "${dup[1]}" — ${dup[2]} shadowed by ${dup[3]}`;

  // Vite's code-splitting warning is ~400 chars of "imported by A, B, C…";
  // the actionable part is which module and why it didn't split.
  const dyn = s.match(
    /^(.+?) is dynamically imported by .+? but also statically imported by .+?, dynamic import will not move module into another chunk\.$/,
  );
  if (dyn) {
    return `${dyn[1]} is both dynamically and statically imported — won't split into its own chunk`;
  }

  // Drop a trailing `, imported by "<path>"` when that path is already named
  // earlier in the message — it just eats the truncation budget.
  const imp = s.match(/^(.*),\s*imported by "([^"]+)"\.$/);
  if (imp && imp[1].includes(imp[2])) {
    return truncate(`${imp[1]}.`, 150);
  }

  return truncate(s, 150);
}

// ── output stats ─────────────────────────────────────────────────────────────

function outputStats(dir) {
  let files = 0;
  let bytes = 0;
  const walk = (current) => {
    let entries;
    try {
      entries = fs.readdirSync(current, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      const full = path.join(current, entry.name);
      if (entry.isDirectory()) walk(full);
      else if (entry.isFile()) {
        files++;
        try {
          bytes += fs.statSync(full).size;
        } catch {
          /* raced with cleanup */
        }
      }
    }
  };
  walk(dir);
  return { files, bytes };
}

function readRouteCount() {
  const raw = safeRead(LOG);
  if (raw) {
    const m = stripAnsi(raw).match(/Prerendered (\d+) routes?/i);
    if (m) return Number(m[1]);
  }
  try {
    return fs.readdirSync(OUTPUT_DIR).filter((f) => f.endsWith('.html')).length;
  } catch {
    return 0;
  }
}
