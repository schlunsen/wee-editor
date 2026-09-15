#!/usr/bin/env node
/**
 * Friendly `go test` wrapper.
 *
 * `go test -v ./...` on this repo is ~6,000 lines of `=== RUN` / `--- PASS`
 * noise. This runs the same thing, keeps every line in a log file, and prints a
 * one-screen summary: how many packages, how many tests, what failed, and
 * exactly which assertions broke.
 *
 * Usage:
 *   node scripts/go-test.mjs                       # all packages, summarised
 *   node scripts/go-test.mjs ./internal/server     # one package
 *   node scripts/go-test.mjs --coverage            # + coverage.out / coverage.html
 *   node scripts/go-test.mjs --verbose             # stream everything live
 *   node scripts/go-test.mjs --show-log            # print the last test log
 *   node scripts/go-test.mjs -- -run TestFoo -count=1   # pass flags to go test
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
  formatDuration,
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
const LOG = logPath('go-test');
const COVER_PROFILE = path.join(ROOT, 'coverage.out');
const COVER_HTML = path.join(ROOT, 'coverage.html');

// Positional args are package patterns; anything after `--` goes to `go test`
// untouched, so `-run TestFoo` (which takes a value) is never misparsed.
const argv = opts.argv;
const sep = argv.indexOf('--');
const beforeSep = sep >= 0 ? argv.slice(0, sep) : argv;
const passthrough = sep >= 0 ? argv.slice(sep + 1) : [];

const WRAPPER_FLAGS = new Set([
  '--verbose',
  '-v',
  '--show-log',
  '--no-warnings',
  '--quiet',
  '-q',
  '--coverage',
  '-c',
]);

const packages = beforeSep.filter((a) => !WRAPPER_FLAGS.has(a) && !a.startsWith('-'));
const testing = packages.length ? packages : ['./...'];
const coverage = opts.has('--coverage', '-c') || process.env.WEE_COVERAGE === '1';

const spinnerRef = { current: null };
const pkgs = { ok: [], fail: [], empty: [] };
const tests = { pass: 0, fail: 0, skip: 0 };
const failures = [];
const panics = [];
let cached = 0;

main().catch((err) => {
  console.error(`${red('✗')} go-test wrapper crashed: ${err?.stack || err}`);
  process.exit(1);
});

async function main() {
  if (opts.showLog) return showLog('go-test');

  if (!opts.quiet) header('🧪 Running tests', `(${testing.join(' ')})`);

  const args = ['test', '-v'];
  if (coverage) args.push('-coverprofile=coverage.out', '-covermode=atomic');
  args.push(...passthrough, ...testing);

  const { code, elapsed } = await runStep(`go test ${testing.join(' ')}`, 'go', args, {
    cwd: ROOT,
    logPath: LOG,
    verbose: opts.verbose,
    onSpinner: (s) => {
      spinnerRef.current = s;
    },
    onLine: collect,
  });

  report(code, elapsed);

  if (coverage && code === 0) await reportCoverage();

  process.exit(code === 0 ? 0 : 1);
}

// ── output parsing ───────────────────────────────────────────────────────────

function collect(line) {
  // `ok   pkg  1.2s` — and with -coverprofile, `ok  pkg  1.2s  coverage: 14.2% of
  // statements`, so the tail after the name is free-form and must not be
  // required to match. `?  pkg  [no test files]` and `FAIL pkg [build failed]`
  // land in the same shape.
  const pkg = line.match(/^(ok|FAIL|\?)\s+(\S+)(.*)$/);
  if (pkg) {
    const [, status, name, rest] = pkg;
    const note = rest.replace(/\(cached\)/, '').replace(/^\s*[\d.]+s\s*/, '').trim();
    if (status === 'ok') {
      if (/\(cached\)/.test(rest)) cached++;
      pkgs.ok.push({ name, note });
    } else if (status === 'FAIL') {
      pkgs.fail.push({ name, note });
    } else {
      pkgs.empty.push(name);
    }
  }

  const result = line.match(/^\s*--- (PASS|FAIL|SKIP):\s+(\S+)\s+\(/);
  if (result) {
    const [, status, name] = result;
    if (status === 'PASS') tests.pass++;
    else if (status === 'SKIP') tests.skip++;
    else {
      tests.fail++;
      // Subtests are reported after their parent; both are worth naming.
      failures.push(name);
    }
  }

  if (line.startsWith('panic:') || /^\s*panic:/.test(line)) panics.push(line.trim());

  // Live progress beats a motionless spinner on a suite this size.
  const seen = tests.pass + tests.fail + tests.skip;
  if (spinnerRef.current && seen > 0 && seen % 25 === 0) {
    spinnerRef.current.setLabel(`go test ${testing.join(' ')} (${seen.toLocaleString('en-US')} tests)`);
  }
}

// ── summary ──────────────────────────────────────────────────────────────────

function report(code, elapsed) {
  const ran = pkgs.ok.length + pkgs.fail.length;
  const bits = [
    `${ran} package${ran === 1 ? '' : 's'}`,
    `${tests.pass.toLocaleString('en-US')} test${tests.pass === 1 ? '' : 's'}`,
  ];
  if (tests.skip) bits.push(`${tests.skip} skipped`);
  if (cached) bits.push(`${cached} cached`);
  if (pkgs.empty.length) bits.push(`${pkgs.empty.length} without tests`);
  bits.push(formatDuration(elapsed));

  if (code === 0) {
    console.log(`${green('✅')} ${bold('Tests passed')}  ${dim(bits.join(' · '))}`);
  } else {
    console.log(
      `${red('✗')} ${bold('Tests failed')}  ${dim(
        `${tests.fail} failing in ${pkgs.fail.length} package${pkgs.fail.length === 1 ? '' : 's'}`,
      )}`,
    );
    console.log(`   ${dim(bits.join(' · '))}`);
  }
  console.log('');

  if (code !== 0) printFailures();

  if (code === 0 && tests.skip) {
    console.log(`   ${dim(`ℹ  ${tests.skip} skipped — see the log for names`)}`);
    console.log('');
  }

  console.log(`   ${dim(`full log: ${rel(LOG)}`)}`);
  if (code !== 0) {
    console.log(`   ${dim(`re-run one package: ${cyan(`just test ${shortPkg(pkgs.fail[0]?.name)}`)}`)}`);
  }
  console.log('');
}

/** `github.com/schlunsen/wee-editor/internal/database` → `./internal/database`. */
function shortPkg(name) {
  if (!name) return './...';
  const marker = '/internal/';
  const i = name.indexOf(marker);
  return i === -1 ? name : `.${name.slice(i)}`;
}

function printFailures() {
  console.log(`   ${yellow('⚠')}  ${bold('Failing tests')}`);
  for (const name of [...new Set(failures)].slice(0, 15)) console.log(`      ${red('•')} ${name}`);
  if (failures.length > 15) console.log(`      ${dim(`… +${failures.length - 15} more in the log`)}`);
  console.log('');

  if (pkgs.fail.length) {
    console.log(`   ${yellow('⚠')}  ${bold('Failing packages')}`);
    for (const p of pkgs.fail) {
      console.log(`      ${red('•')} ${p.name}${p.note ? ` ${dim(`[${p.note}]`)}` : ''}`);
    }
    console.log('');
  }

  const region = failureRegion();
  if (region.length) {
    console.log(`   ${yellow('⚠')}  ${bold('Failure detail')}`);
    for (const line of region) console.log(`      ${line}`);
    console.log('');
  }

  if (panics.length) {
    console.log(`   ${yellow('⚠')}  ${bold('Panics')}`);
    for (const line of [...new Set(panics)].slice(0, 5)) console.log(`      ${red('•')} ${line}`);
    console.log('');
  }
}

/**
 * Pull the first failing test's output out of the log.
 *
 * Go is inconsistent about ordering: a plain test prints its assertion *before*
 * its `--- FAIL:` line, while a failing subtest is marked first and its
 * assertion follows under the parent's marker. So rather than scanning forward
 * from a marker, take the whole region — from the `=== RUN` that opened the
 * test (or, for a package that failed to compile, the errors above it) through
 * to the package verdict.
 */
function failureRegion(maxLines = 20) {
  const raw = safeRead(LOG);
  if (!raw) return [];

  const lines = raw.split(/\r?\n/);
  const firstFail = lines.findIndex((l) => /^\s*--- FAIL:/.test(l));
  if (firstFail === -1) return [];

  // Back up to whatever opened this failure.
  let start = firstFail;
  for (let i = firstFail; i >= 0; i--) {
    if (/^=== RUN/.test(lines[i])) {
      start = i;
      break;
    }
  }
  if (start === firstFail) {
    // No `=== RUN` — a build failure. Walk back over the compiler errors.
    while (start > 0 && lines[start - 1].trim() !== '' && !/^(ok|FAIL|\?)\s/.test(lines[start - 1])) {
      start--;
    }
  }

  const out = [];
  for (let i = start; i < lines.length && out.length < maxLines; i++) {
    const line = lines[i];
    if (i > firstFail && /^(ok|FAIL|\?)\s/.test(line)) break;
    if (line.trim() === '') break;
    out.push(line.replace(/^ {4}/, '').trimEnd());
  }
  return out;
}

// ── coverage ─────────────────────────────────────────────────────────────────

async function reportCoverage() {
  if (!fs.existsSync(COVER_PROFILE)) return;

  let total = null;
  const { code } = await runStep('go tool cover -func', 'go', ['tool', 'cover', `-func=${COVER_PROFILE}`], {
    cwd: ROOT,
    logPath: logPath('go-cover'),
    verbose: opts.verbose,
    onLine: (line) => {
      const m = line.match(/^total:\s+\(statements\)\s+([\d.]+%)/);
      if (m) total = m[1];
    },
  });

  if (code === 0) {
    await runStep('go tool cover -html', 'go', ['tool', 'cover', `-html=${COVER_PROFILE}`, '-o', COVER_HTML], {
      cwd: ROOT,
      logPath: logPath('go-cover'),
      append: true,
      verbose: opts.verbose,
    });
  }

  if (total) console.log(`${green('📊')} ${bold(`Coverage ${total}`)}  ${dim('statements')}`);
  console.log(`   ${dim(`HTML: ${rel(COVER_HTML)}   Data: ${rel(COVER_PROFILE)}`)}`);
  console.log('');
}
