import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import YAML from 'yaml';
import { checkActionTables } from './lib/action-tables.mjs';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const page = fs.readFileSync(path.join(repoRoot, 'docs', 'reference', 'github-actions.md'), 'utf8');
const actionsDirectory = path.join(repoRoot, '.github', 'actions');
const problems = [];

for (const name of fs.readdirSync(actionsDirectory)) {
  const file = path.join(actionsDirectory, name, 'action.yml');
  if (!fs.existsSync(file)) continue;
  const metadata = YAML.parse(fs.readFileSync(file, 'utf8'));
  const start = page.indexOf(`## ${name}`);
  if (start === -1) {
    problems.push(`reference page has no "## ${name}" section`);
    continue;
  }
  const next = page.indexOf('\n## ', start + 3);
  const section = page.slice(start, next === -1 ? undefined : next);
  const inputsStart = section.indexOf('### Inputs');
  const outputsStart = section.indexOf('### Outputs');
  const inputsText = inputsStart === -1 ? '' : section.slice(inputsStart, outputsStart === -1 ? undefined : outputsStart);
  const outputsText = outputsStart === -1 ? '' : section.slice(outputsStart, section.indexOf('\n### ', outputsStart + 3) === -1 ? undefined : section.indexOf('\n### ', outputsStart + 3));
  problems.push(...checkActionTables(name, metadata, inputsText, outputsText));
}

if (problems.length > 0) {
  console.error(`Action documentation does not match action.yml:\n- ${problems.join('\n- ')}`);
  process.exit(1);
}
console.log('Action input/output names and nonempty defaults match action.yml');
