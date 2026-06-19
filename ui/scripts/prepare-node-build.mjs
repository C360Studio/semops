import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { join } from 'node:path';

const buildDir = 'build';
const chunkDir = join(buildDir, 'server', 'chunks');

function mirrorStaticDir(name) {
  const source = join(buildDir, name);
  const target = join(chunkDir, name);
  if (!existsSync(source)) {
    return;
  }
  // adapter-node serves static assets from the bundled handler chunk directory
  // in this toolchain; keep requests on SvelteKit's own static middleware so
  // immutable cache headers and compressed variants remain framework-owned.
  rmSync(target, { force: true, recursive: true });
  mkdirSync(chunkDir, { recursive: true });
  cpSync(source, target, { recursive: true });
}

mirrorStaticDir('client');
mirrorStaticDir('prerendered');
