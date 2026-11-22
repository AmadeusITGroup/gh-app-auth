import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
try {
  execFileSync('go', ['run', './website/scripts/checkconfig'], {
    cwd: repoRoot,
    stdio: 'inherit',
  });
} catch {
  process.exit(1);
}
