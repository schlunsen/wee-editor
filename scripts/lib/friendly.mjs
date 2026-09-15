/**
 * Shared plumbing for wee's "friendly" command wrappers.
 *
 * Every wrapper follows the same shape: run a noisy command with a spinner,
 * write the complete output to a log file, then print a short summary of the
 * bits worth reading (warnings, errors, stats). Nothing is ever discarded —
 * the full stream is always on disk and one flag away.
 *
 *   --verbose      stream the complete output live instead of summarising
 *   --show-log     print the last log and exit
 *   --no-warnings  build quietly and skip the warning digest
 *
 * Env:
 *   WEE_VERBOSE=1        same as --verbose
 *   WEE_LOG_DIR=<path>   override the log directory
 *   NO_COLOR=1           disable colour
 *
 * CommonJS-free ES modules, run by the `node` that npm already requires, so the
 * wrappers behave identically on macOS, Linux and Windows (macOS still ships
 * bash 3.2).
 */

import fs from 'node:fs';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';

export const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
/** Repo root — `scripts/lib/` is two levels down. */
export const ROOT = path.resolve(SCRIPT_DIR, '..', '..');
export const FRONTEND_DIR = path.join(ROOT, 'internal', 'server', 'frontend');
export const LOG_DIR = process.env.WEE_LOG_DIR || path.join(ROOT, 'dist', 'logs');

// ── colour ───────────────────────────────────────────────────────────────────

const ESC = '\x1b';
export const IS_TTY = Boolean(process.stdout.isTTY);
export const COLOR =
  IS_TTY && !process.env.NO_COLOR && process.env.TERM !== 'dumb' && process.env.FORCE_COLOR !== '0';

const paint = (code) => (s) => (COLOR ? `${ESC}[${code}m${s}${ESC}[0m` : String(s));
export const bold = paint('1');
export const dim = paint('2');
export const red = paint('31');
export const green = paint('32');
export const yellow = paint('33');
export const cyan = paint('36');

// ── arguments ────────────────────────────────────────────────────────────────

/** Parse the flags every wrapper shares. Script-specific flags are read separately. */
export function cli(argv = process.argv.slice(2)) {
  const has = (...names) => names.some((n) => argv.includes(n));
  const verbose = has('--verbose', '-v') || process.env.WEE_VERBOSE === '1';
  return {
    argv,
    verbose,
    quiet: has('--quiet', '-q'),
    showLog: has('--show-log'),
    noWarnings: has('--no-warnings') || has('--quiet', '-q'),
    has,
    /** `--flag value` → value, else undefined. */
    value(name) {
      const i = argv.indexOf(name);
      return i >= 0 ? argv[i + 1] : undefined;
    },
  };
}

// ── paths & formatting ───────────────────────────────────────────────────────

/** Repo-relative when inside the repo, absolute otherwise (`../../..` is useless). */
export const rel = (p) => {
  const r = path.relative(ROOT, p);
  return !r || r.startsWith('..') ? p : r;
};

export function logPath(name) {
  return path.join(LOG_DIR, `${name}.log`);
}

export function formatBytes(n) {
  if (n >= 1024 ** 3) return `${(n / 1024 ** 3).toFixed(1)} GB`;
  if (n >= 1024 ** 2) return `${(n / 1024 ** 2).toFixed(1)} MB`;
  if (n >= 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${n} B`;
}

export function formatDuration(ms) {
  return ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${ms}ms`;
}

export function fileSize(p) {
  try {
    return fs.statSync(p).size;
  } catch {
    return 0;
  }
}

export function stripAnsi(s) {
  return s
    .replace(new RegExp(`${ESC}\\][^\\x07]*(?:\\x07|${ESC}\\\\)`, 'g'), '') // OSC
    .replace(new RegExp(`${ESC}\\[[0-9;?]*[ -/]*[@-~]`, 'g'), ''); // CSI
}

export function safeRead(p) {
  try {
    return fs.readFileSync(p, 'utf8');
  } catch {
    return null;
  }
}

/**
 * Turn absolute paths into repo-relative ones so output is short, readable and
 * stable across machines. Longest prefix first, so a frontend path collapses to
 * `app/…` rather than `internal/server/frontend/app/…`.
 */
export function relativize(s, bases = [FRONTEND_DIR, ROOT]) {
  let out = s;
  for (const base of [...bases].sort((a, b) => b.length - a.length)) {
    const escaped = base.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    out = out.replace(new RegExp(`${escaped}/`, 'g'), '');
  }
  return out;
}

export function truncate(s, max) {
  return s.length > max ? `${s.slice(0, max - 1).trimEnd()}…` : s;
}

// ── step runner ──────────────────────────────────────────────────────────────

/**
 * Run one command, hiding its output behind a spinner (or streaming it with
 * `verbose: true`), while writing everything to `logPath`.
 *
 * `onLine` receives each complete stdout/stderr line as it arrives, so callers
 * can aggregate (count tests, collect failures) without ever holding the whole
 * output in memory — `go test -v` on this repo is tens of thousands of lines.
 *
 * Resolves `{ code, logPath, elapsed }`.
 */
export function runStep(label, cmd, cmdArgs, opts = {}) {
  const {
    cwd = ROOT,
    logPath: log = logPath('build'),
    verbose = false,
    env = {},
    onLine,
    onSpinner,
    append = false,
    echo = false,
  } = opts;

  fs.mkdirSync(path.dirname(log), { recursive: true });
  if (append) {
    fs.appendFileSync(log, `\n\n──── ${label} — ${cmd} ${cmdArgs.join(' ')}\n`);
  }
  const out = fs.createWriteStream(log, { flags: append ? 'a' : 'w' });
  const started = Date.now();

  if (verbose) {
    console.log(dim(`   $ ${cmd} ${cmdArgs.join(' ')}`));
    if (echo && cmdArgs.length) console.log(dim(`     → ${cmdArgs[cmdArgs.length - 1]}`));
  }

  const spawnOpts = {
    cwd,
    shell: process.platform === 'win32',
    stdio: ['ignore', 'pipe', 'pipe'],
    env: { ...process.env, ...env },
  };

  const spinner = verbose ? null : startSpinner(label);
  if (spinner && onSpinner) onSpinner(spinner);

  return new Promise((resolve) => {
    const child = spawn(cmd, cmdArgs, spawnOpts);

    let carry = '';
    const feed = (chunk, sink) => {
      const text = chunk.toString();
      if (sink) sink.write(text);
      if (!onLine) return;
      carry += text;
      const lines = carry.split(/\r?\n/);
      carry = lines.pop() ?? '';
      for (const line of lines) onLine(stripAnsi(line));
    };

    child.stdout.on('data', (d) => {
      if (verbose) process.stdout.write(d);
      feed(d, out);
    });
    child.stderr.on('data', (d) => {
      if (verbose) process.stderr.write(d);
      feed(d, out);
    });

    const finish = (code) => {
      if (carry && onLine) onLine(stripAnsi(carry));
      out.end(() => {
        if (spinner) spinner.stop(code === 0 ? 'done' : 'fail');
        resolve({ code: code ?? 1, logPath: log, elapsed: Date.now() - started });
      });
    };

    child.on('close', finish);
    child.on('error', () => finish(1));

    const onSignal = () => {
      try {
        child.kill('SIGTERM');
      } catch {
        /* already gone */
      }
      if (spinner) spinner.stop('fail');
      process.exit(130);
    };
    process.once('SIGINT', onSignal);
    process.once('SIGTERM', onSignal);
  });
}

export function startSpinner(label) {
  const frames = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];
  let i = 0;
  let timer = null;
  let text = label;

  const draw = () => {
    process.stdout.write(`\r${ESC}[K${cyan(frames[i++ % frames.length])} ${text}${dim('…')}`);
  };

  if (IS_TTY) {
    draw();
    timer = setInterval(draw, 80);
  } else {
    console.log(`${dim('·')} ${label}...`);
  }

  return {
    /** Replace the spinner text — used to report live progress mid-run. */
    setLabel(next) {
      text = next;
    },
    stop(state) {
      if (timer) clearInterval(timer);
      timer = null;
      if (IS_TTY) process.stdout.write(`\r${ESC}[K`);
      const mark = state === 'done' ? green('✓') : red('✗');
      console.log(`${mark} ${text}`);
    },
  };
}

// ── headings ─────────────────────────────────────────────────────────────────

export function header(text, sub = '') {
  console.log(`\n${bold(text)}${sub ? ` ${dim(sub)}` : ''}\n`);
}

// ── failure path ─────────────────────────────────────────────────────────────

/**
 * Report a failed step: message, a tail of the log for context, and the paths
 * that lead to the full story. Exits the process.
 */
export function fail(code, message, opts = {}) {
  const { logPath: log = logPath('build'), tail = 30, hint = '' } = opts;

  console.log('');
  console.error(`${red('✗')} ${bold(message)} ${dim(`(exit ${code})`)}`);
  console.error('');

  const raw = safeRead(log);
  if (raw) {
    const lines = raw.split(/\r?\n/).filter((l) => l.trim() !== '');
    const rule = '─'.repeat(42);
    console.error(dim(`   ── last ${tail} log lines ${rule}`));
    for (const line of lines.slice(-tail)) console.error(`   ${relativize(line)}`);
    console.error(dim(`   ${'─'.repeat(61)}`));
  }

  console.error('');
  if (hint) console.error(`   ${hint}`);
  console.error(`   full log: ${rel(log)}`);
  console.error('');
  process.exit(code || 1);
}

// ── warning digest ───────────────────────────────────────────────────────────

/**
 * Print the compact warning digest shared by every wrapper.
 *
 * `actionable` entries are shown (capped); `benign` ones are counted and named
 * but never silently dropped. With `show: false` the whole digest is skipped and
 * only the log path is printed.
 */
export function printWarnings({ actionable = [], benign = [], log = null, show = true, cap = 8 } = {}) {
  if (!show) {
    if (log && actionable.length) console.log(`   ${dim(`full log: ${rel(log)}`)}`);
    return;
  }

  if (actionable.length) {
    console.log(
      `   ${yellow('⚠')}  ${bold(String(actionable.length))} warning${actionable.length === 1 ? '' : 's'}`,
    );
    for (const w of actionable.slice(0, cap)) console.log(`      ${dim('•')} ${w}`);
    if (actionable.length > cap) {
      console.log(`      ${dim(`… +${actionable.length - cap} more in the log`)}`);
    }
  } else {
    console.log(`   ${green('✓')} ${dim('no warnings')}`);
  }

  if (benign.length) {
    const named = benign.slice(0, 2).join(', ');
    const more = benign.length > 2 ? `, +${benign.length - 2} more` : '';
    console.log(`   ${dim(`ℹ  ${benign.length} informational — ${named}${more}`)}`);
  }

  if (log && actionable.length) console.log(`\n   ${dim(`full log: ${rel(log)}`)}`);
}

// ── log viewer ───────────────────────────────────────────────────────────────

/** Print a log file (ANSI stripped) or list what is available. */
export function showLog(name) {
  const target = name
    ? /[/\\]|\.log$/.test(name)
      ? path.resolve(ROOT, name)
      : logPath(name)
    : mostRecentLog();

  if (!target || !fs.existsSync(target)) {
    console.log(
      `${yellow('⚠')}  No log at ${rel(target || LOG_DIR)}${name ? '' : ' yet'} — run something first.\n`,
    );
    listLogs();
    process.exit(1);
  }
  process.stdout.write(stripAnsi(safeRead(target) || ''));
}

export function listLogs() {
  let entries;
  try {
    entries = fs.readdirSync(LOG_DIR).filter((f) => f.endsWith('.log'));
  } catch {
    console.log(dim(`   (no logs yet in ${rel(LOG_DIR)})`));
    return;
  }
  if (!entries.length) {
    console.log(dim(`   (no logs yet in ${rel(LOG_DIR)})`));
    return;
  }
  const rows = entries
    .map((f) => ({ f, p: path.join(LOG_DIR, f), m: fileSize(path.join(LOG_DIR, f)) }))
    .sort((a, b) => b.m - a.m);
  console.log(bold('   Available logs'));
  for (const r of rows) {
    console.log(`   ${dim('•')} ${r.f.replace(/\.log$/, '').padEnd(16)} ${dim(formatBytes(r.m))}`);
  }
  console.log(dim(`\n   view one with: just logs <name>`));
}

function mostRecentLog() {
  let entries;
  try {
    entries = fs.readdirSync(LOG_DIR).filter((f) => f.endsWith('.log'));
  } catch {
    return null;
  }
  let best = null;
  for (const f of entries) {
    const p = path.join(LOG_DIR, f);
    const t = (() => {
      try {
        return fs.statSync(p).mtimeMs;
      } catch {
        return 0;
      }
    })();
    if (!best || t > best.t) best = { p, t };
  }
  return best?.p ?? null;
}
