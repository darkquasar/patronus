// Invented local helper fixtures, separate from the historical 01..09 model probes.
import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync, readFileSync, rmSync, symlinkSync, statSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
const scripts = fileURLToPath(new URL('../scripts/', import.meta.url));
function fixture(t) {
  const dir = mkdtempSync(join(tmpdir(), 'pi-plan-helpers-'));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  const plan = join(dir, 'invented-plan.md');
  writeFileSync(plan, '# Invented plan\n\n## Task 1: first\nfirst behavior\n```md\n## Task 2: fenced decoy\n```\n## Task 2: second\nsecond behavior\n');
  return { dir, plan };
}
function run(command, args, cwd) {
  const result = spawnSync(command, args, { cwd, encoding: 'utf8', env: { PATH: process.env.PATH, HOME: cwd, GIT_CONFIG_NOSYSTEM: '1', GIT_CONFIG_GLOBAL: '/dev/null', GIT_AUTHOR_NAME: 'Fixture', GIT_AUTHOR_EMAIL: 'fixture@example.invalid', GIT_COMMITTER_NAME: 'Fixture', GIT_COMMITTER_EMAIL: 'fixture@example.invalid' } });
  assert.equal(result.error, undefined);
  return result;
}
test('task-brief extracts actual task boundaries, preserves fenced text and rejects absent tasks', t => {
  const { dir, plan } = fixture(t); const output = join(dir, 'brief.md');
  const script = join(scripts, 'task-brief');
  assert.ok(statSync(script).mode & 0o111);
  assert.equal(run(script, [plan, '1', output], dir).status, 0);
  assert.equal(readFileSync(output, 'utf8'), '## Task 1: first\nfirst behavior\n```md\n## Task 2: fenced decoy\n```\n');
  const failed = run(script, [plan, '9', output], dir);
  assert.equal(failed.status, 3); assert.match(failed.stderr, /task 9 not found/);
});
test('workspace resolves a symlinked plan to the same scratch path and keeps it ignored', t => {
  const { dir, plan } = fixture(t); const link = join(dir, 'other-name.md'); symlinkSync(plan, link);
  const script = join(scripts, 'sdd-workspace');
  const direct = run(script, [plan], dir); const alias = run(script, [link], dir);
  assert.equal(direct.status, 0); assert.equal(alias.status, 0); assert.equal(alias.stdout, direct.stdout);
  assert.equal(readFileSync(join(direct.stdout.trim(), '.gitignore'), 'utf8'), '*\n');
});
test('review-package retains all commits from recorded base, not only HEAD~1', t => {
  const { dir, plan } = fixture(t);
  const git = (...args) => { const r = run('git', args, dir); assert.equal(r.status, 0, r.stderr); return r.stdout.trim(); };
  git('init', '--quiet'); git('add', 'invented-plan.md'); git('commit', '--quiet', '-m', 'fixture base');
  const base = git('rev-parse', 'HEAD');
  writeFileSync(join(dir, 'first.txt'), 'first change'); git('add', 'first.txt'); git('commit', '--quiet', '-m', 'fixture first');
  writeFileSync(join(dir, 'second.txt'), 'second change'); git('add', 'second.txt'); git('commit', '--quiet', '-m', 'fixture second');
  const head = git('rev-parse', 'HEAD'); const output = join(dir, 'review.diff');
  const result = run(join(scripts, 'review-package'), [plan, base, head, output], dir);
  assert.equal(result.status, 0); assert.match(result.stdout, /2 commit\(s\)/);
  const bytes = readFileSync(output, 'utf8'); assert.match(bytes, /fixture first/); assert.match(bytes, /fixture second/);
  assert.match(bytes, /\+first change/); assert.match(bytes, /\+second change/);
  assert.equal(run(join(scripts, 'review-package'), [plan, 'absent-ref', head, output], dir).status, 2);
});
