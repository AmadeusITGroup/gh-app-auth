import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { normalizeHelp, parseCommands, renderPage } from './lib/reference-pages.mjs';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const outputDirectory = path.join(repoRoot, 'docs', 'reference', 'cli');
const checkOnly = process.argv.includes('--check');

function buildBinary(workDirectory) {
  const binary = path.join(workDirectory, 'gh-app-auth');
  execFileSync('go', ['build', '-o', binary, '.'], { cwd: repoRoot, stdio: 'inherit' });
  return binary;
}

function cleanEnvironment(workDirectory) {
  const home = path.join(workDirectory, 'home');
  fs.mkdirSync(home, { recursive: true });
  return {
    PATH: process.env.PATH,
    HOME: home,
    GH_CONFIG_DIR: path.join(home, '.config', 'gh'),
    XDG_CONFIG_HOME: path.join(home, '.config'),
    NO_COLOR: '1',
    LANG: 'C',
  };
}

function help(binary, args, env) {
  return normalizeHelp(
    execFileSync(binary, [...args, '--help'], { env, cwd: path.dirname(binary), encoding: 'utf8' }),
  );
}

function capture(binary, env) {
  const root = help(binary, [], env);
  const pages = new Map();
  for (const { name, summary } of parseCommands(root)) {
    const top = help(binary, [name], env);
    const sections = [{ title: name, help: top }];
    for (const child of parseCommands(top)) {
      sections.push({ title: `gh app-auth ${name} ${child.name}`, help: help(binary, [name, child.name], env) });
    }
    pages.set(name, renderPage({ name, summary, sections }));
  }
  return pages;
}

const workDirectory = fs.mkdtempSync(path.join(os.tmpdir(), 'gh-app-auth-reference-'));
try {
  const binary = buildBinary(workDirectory);
  const pages = capture(binary, cleanEnvironment(workDirectory));
  const existing = fs
    .readdirSync(outputDirectory)
    .filter((file) => file.endsWith('.md') && file !== 'index.md');
  const expected = new Set([...pages.keys()].map((name) => `${name}.md`));
  const problems = [];
  for (const file of existing) if (!expected.has(file)) problems.push(`unexpected page ${file}`);
  for (const [name, content] of pages) {
    const file = path.join(outputDirectory, `${name}.md`);
    const current = fs.existsSync(file) ? fs.readFileSync(file, 'utf8') : null;
    if (current !== content) {
      if (checkOnly) problems.push(`stale or missing page ${name}.md`);
      else fs.writeFileSync(file, content);
    }
  }
  if (checkOnly) {
    if (problems.length > 0) {
      console.error(`CLI reference is out of date:\n- ${problems.join('\n- ')}\nRun npm run reference:generate`);
      process.exitCode = 1;
    } else {
      console.log(`CLI reference matches ${pages.size} commands`);
    }
  } else {
    for (const problem of problems) console.warn(problem);
    console.log(`Generated ${pages.size} command pages`);
  }
} finally {
  fs.rmSync(workDirectory, { recursive: true, force: true });
}
