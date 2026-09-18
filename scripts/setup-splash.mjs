#!/usr/bin/env node
// Explicit optional install. Normal builds never fetch or compile third-party code.
import fs from 'node:fs';
import path from 'node:path';
import { ROOT, runStep, logPath } from './lib/friendly.mjs';
const revision = '7c4905c50fdd880709c1e8097cbc170970c22f6f';
const tools = path.join(ROOT, 'dist', 'tools');
fs.mkdirSync(tools, { recursive: true });
const source = fs.mkdtempSync(path.join(tools, 'rasterminal-source-'));
const binary = process.platform === 'win32' ? 'rasterminal.exe' : 'rasterminal';
const steps = [
  ['git', ['init', source]],
  ['git', ['-C', source, 'fetch', '--depth', '1', 'https://github.com/PavolUlicny/rasterminal.git', revision]],
  ['git', ['-C', source, 'checkout', '--detach', 'FETCH_HEAD']],
  ['cmake', ['-S', source, '-B', path.join(source, 'build'), '-DCMAKE_BUILD_TYPE=Release', '-DRASTERMINAL_PORTABLE=ON']],
  ['cmake', ['--build', path.join(source, 'build'), '--config', 'Release', '--parallel', '4', '--target', 'rasterminal']]
];
try {
  for (const [command, args] of steps) {
    const result = await runStep(`Setting up Rasterminal: ${command}`, command, args, { logPath: logPath('rasterminal-setup'), append: true });
    if (result.code) throw new Error(`Setup failed. See ${logPath('rasterminal-setup')}`);
  }
  const output = [path.join(source, 'build', binary), path.join(source, 'build', 'Release', binary)].find(p => fs.existsSync(p));
  if (!output) throw new Error('Rasterminal build output not found');
  fs.copyFileSync(output, path.join(tools, binary));
  if (process.platform !== 'win32') fs.chmodSync(path.join(tools, binary), 0o755);
  console.log('Horse splash ready for just build. Disable with WEE_SPLASH=0.');
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
// Retain the source, license and third-party notices beside the local binary.
