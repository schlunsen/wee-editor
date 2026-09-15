#!/usr/bin/env node
/**
 * Friendly lint wrapper.
 *
 * Uses golangci-lint when it is installed (there is a .golangci.yml in the
 * repo) and falls back to `go vet ./...` when it is not. Either way you get a
 * count, a grouped list of findings, and the full report in a log file instead
 * of a wall of file:line pairs.
 *
 * Usage:
 *   node scripts/go-lint.mjs            # lint + summary
 *   node scripts/go-lint.mjs --verbose  # stream the complete output
 *   node scripts/go-lint.mjs --no-fail  # exit 0 even when issues are found
 *   node scripts/go-lint.mjs --show-log
 *
 * Env: see lib/friendly.mjs
 */

import { spawnSync } from 'node:child_process';
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
  showLog,
  truncate,
  yellow,
} from './lib/friendly.mjs';

const opts = cli();
const LOG = logPath('go-lint');
const strict = !opts.has('--no-fail');

const useGolangci = has('golangci-lint');

main().catch((err) => {
  console.error(`${red('✗')} go-lint wrapper crashed: ${err?.stack || err}`);
  process.exit(1);
});

async function main() {
  if (opts.showLog) return showLog('go-lint');

  if (!opts.quiet) {
    header('🔍 Linting', `(${useGolangci ? 'golangci-lint' : 'go vet — golangci-lint not installed'})`);
  }

  const tool = useGolangci ? 'golangci-lint' : 'go';
  const args = useGolangci ? ['run', '--color=never'] : ['vet', './...'];

  const findings = [];
  let currentPkg = null;

  const { code } = await runStep(args.join(' '), tool, args, {
    cwd: ROOT,
    logPath: LOG,
    verbose: opts.verbose,
    onLine: (line) => {
      if (line.startsWith('# ')) {
        currentPkg = line.slice(2).trim(); // go vet package header
        return;
      }
      const m = line.match(/^(.+?\.go):(\d+):(\d+):\s*(.+)$/);
      if (!m) return;
      const [, file, lineNo, col, rest] = m;
      const linter = rest.match(/\((\w[\w-]*)\)\s*$/)?.[1] || null;
      findings.push({
        file: relativizePath(file),
        where: `${lineNo}:${col}`,
        linter: linter || (currentPkg ? 'vet' : 'lint'),
        message: truncate(linter ? rest.replace(/\s*\(\w[\w-]*\)\s*$/, '') : rest, 120),
      });
    },
  });

  // A crash, a bad config, or a compile error — the tool itself did not run.
  if (code !== 0 && !findings.length) {
    return fail(code, `${tool} failed to run`, { logPath: LOG });
  }

  report(findings, code);
  process.exit(findings.length && strict ? 1 : 0);
}

function report(findings, code) {
  if (!findings.length) {
    console.log(`${green('✅')} ${bold('No issues')}`);
    console.log('');
    console.log(`   ${dim(`full log: ${rel(LOG)}`)}`);
    console.log('');
    return;
  }

  const byLinter = new Map();
  for (const f of findings) byLinter.set(f.linter, (byLinter.get(f.linter) || 0) + 1);
  const files = new Set(findings.map((f) => f.file));

  console.log(
    `${yellow('⚠')}  ${bold(`${findings.length} issue${findings.length === 1 ? '' : 's'}`)} ${dim(
      `in ${files.size} file${files.size === 1 ? '' : 's'}`,
    )}`,
  );
  console.log(
    `   ${dim([...byLinter.entries()].sort((a, b) => b[1] - a[1]).map(([l, n]) => `${l} ${n}`).join(' · '))}`,
  );
  console.log('');

  const cap = 12;
  for (const f of findings.slice(0, cap)) {
    console.log(`   ${dim('•')} ${f.file}:${f.where}  ${f.message} ${dim(`(${f.linter})`)}`);
  }
  if (findings.length > cap) {
    console.log(`   ${dim(`… +${findings.length - cap} more in the log`)}`);
  }

  console.log('');
  console.log(`   ${dim(`full log: ${rel(LOG)}`)}`);
  if (strict) console.log(`   ${dim(`pass --no-fail to exit 0 anyway`)}`);
  console.log('');
}

function relativizePath(p) {
  return p.startsWith(ROOT) ? p.slice(ROOT.length + 1) : p;
}

function has(bin) {
  const probe = spawnSync(bin, ['--version'], { stdio: 'ignore' });
  return !probe.error;
}
