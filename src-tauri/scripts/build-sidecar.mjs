#!/usr/bin/env node

/**
 * Build the Go wee binary as a Tauri sidecar.
 *
 * This script:
 * 1. Builds the Nuxt frontend (generate static files)
 * 2. Builds the Go binary with the embedded frontend
 * 3. Copies it to src-tauri/sidecars/ with the correct platform triple
 */

import { execSync } from 'child_process';
import { copyFileSync, mkdirSync, existsSync } from 'fs';
import { join, resolve } from 'path';
import { platform, arch } from 'os';

const ROOT = resolve(import.meta.dirname, '..', '..');
const SIDECARS_DIR = resolve(import.meta.dirname, '..', 'sidecars');

// Map Node.js platform/arch to Rust target triple
function getTargetTriple() {
  const archMap = { x64: 'x86_64', arm64: 'aarch64', ia32: 'i686' };
  const osMap = {
    darwin: 'apple-darwin',
    linux: 'unknown-linux-gnu',
    win32: 'pc-windows-msvc',
  };

  const rustArch = archMap[arch()] || arch();
  const rustOs = osMap[platform()] || platform();
  return `${rustArch}-${rustOs}`;
}

function run(cmd, opts = {}) {
  console.log(`  → ${cmd}`);
  execSync(cmd, { stdio: 'inherit', cwd: ROOT, ...opts });
}

async function main() {
  const triple = getTargetTriple();
  const ext = platform() === 'win32' ? '.exe' : '';
  const sidecarName = `wee-server-${triple}${ext}`;

  console.log(`\n🔧 Building Wee sidecar for ${triple}\n`);

  // Step 1: Build frontend
  console.log('📦 Step 1: Building Nuxt frontend...');
  const frontendDir = join(ROOT, 'internal', 'server', 'frontend');
  if (!existsSync(join(frontendDir, 'node_modules'))) {
    run('npm install', { cwd: frontendDir });
  }
  run('npm run generate', { cwd: frontendDir });

  // Step 2: Build Go binary
  console.log('\n🦫 Step 2: Building Go binary...');
  const goBinary = join(ROOT, `wee${ext}`);
  run(`go build -o wee${ext} ./cmd/wee`);

  // Step 3: Copy to sidecars directory
  console.log('\n📋 Step 3: Copying to sidecars directory...');
  mkdirSync(SIDECARS_DIR, { recursive: true });
  const dest = join(SIDECARS_DIR, sidecarName);
  copyFileSync(goBinary, dest);
  console.log(`  → ${dest}`);

  console.log(`\n✅ Sidecar built: ${sidecarName}\n`);
}

main().catch((err) => {
  console.error('❌ Build failed:', err.message);
  process.exit(1);
});
