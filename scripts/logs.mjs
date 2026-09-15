#!/usr/bin/env node
/**
 * List the wrapper log files, or print one of them.
 *
 * Every friendly wrapper writes its complete output under dist/logs/, so the
 * answer to "what did that build actually say?" is always one command away.
 *
 * Usage:
 *   node scripts/logs.mjs                 # list what is available
 *   node scripts/logs.mjs frontend-build  # print one (ANSI stripped)
 *   node scripts/logs.mjs dist/logs/go-test.log
 */

import { LOG_DIR, bold, cli, dim, listLogs, rel, showLog } from './lib/friendly.mjs';

const opts = cli();
const name = opts.argv.find((a) => !a.startsWith('-'));

if (name) {
  showLog(name);
} else {
  console.log(`\n${bold('📄 Build logs')} ${dim(rel(LOG_DIR))}\n`);
  listLogs();
  console.log('');
}
