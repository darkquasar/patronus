// Opt-in installed-runtime check for Pi 0.87.1 and pi-subagents 0.72.1.
// Loads extensions, but never starts a child, prompts a model, or invokes a tool.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const [piRootArg, agentDirArg, cwdArg, outputArg, roleDirArg] = process.argv.slice(2);
assert(piRootArg && agentDirArg && cwdArg && outputArg,
  'usage: node pi-native-resources.mjs PI_PACKAGE_DIR AGENT_DIR PROJECT OUTPUT_JSON [ROLE_DIR]');
const piRoot = resolve(piRootArg);
const agentDir = resolve(agentDirArg);
const cwd = resolve(cwdArg);
const roleDir = roleDirArg ? resolve(roleDirArg) : join(agentDir, 'agents');
const subagentsRoot = join(agentDir, 'npm/node_modules/pi-subagents');
const webRoot = join(agentDir, 'npm/node_modules/pi-web-access');
const digest = bytes => createHash('sha256').update(bytes).digest('hex');
const packages = [[piRoot, '0.87.1'], [subagentsRoot, '0.72.1'], [webRoot, '0.35.0']]
  .map(([root, version]) => {
    const bytes = readFileSync(join(root, 'package.json'));
    const metadata = JSON.parse(bytes);
    assert.equal(metadata.version, version, metadata.name);
    return { name: metadata.name, version, metadataSHA256: digest(bytes) };
  });
const { DefaultResourceLoader } = await import(pathToFileURL(join(piRoot, 'dist/index.js')));
const { discoverAgents } = await import(pathToFileURL(join(subagentsRoot, 'src/agents/agents.js')));
const loader = new DefaultResourceLoader({ cwd, agentDir });
await loader.reload();
const loaded = loader.getExtensions();
assert.deepEqual(loaded.errors, [], 'extension load errors');
const extensions = loaded.extensions.map(extension => ({
  path: extension.path, tools: [...extension.tools.keys()].sort(),
}));
const availableTools = new Set(extensions.flatMap(extension => extension.tools));
for (const tool of ['subagent', 'bg_wait', 'web_search', 'source_check', 'fetch_content', 'get_search_content']) {
  assert(availableTools.has(tool), `missing registered tool ${tool}`);
}
const skills = loader.getSkills();
assert.deepEqual(skills.diagnostics, [], 'skill load diagnostics');
const skillNames = new Set(skills.skills.map(skill => skill.name));
for (const name of ['workflow-research-pi', 'workflow-implement-pi', 'workflow-peer-review-pi']) {
  assert(skillNames.has(name), `missing workflow skill ${name}`);
}
const expectedRoles = readdirSync(roleDir)
  .filter(name => name.startsWith('patronus-') && name.endsWith('-pi.md'))
  .map(name => name.slice(0, -3));
assert.equal(expectedRoles.length, 8, 'core-profile-pi must deploy eight roles');
const discovered = discoverAgents(cwd, 'both').agents;
const roles = expectedRoles.map(name => {
  const role = discovered.find(agent => agent.name === name);
  assert(role, `native discovery missed ${name}`);
  assert.equal(resolve(role.filePath), join(roleDir, `${name}.md`), `${name}: wrong role source`);
  for (const skill of role.skills ?? []) assert(skillNames.has(skill), `${name}: missing skill ${skill}`);
  for (const tool of role.tools ?? []) {
    assert(['read', 'bash', 'write', 'edit'].includes(tool) || availableTools.has(tool),
      `${name}: missing tool ${tool}`);
  }
  return {
    name, skills: role.skills, tools: role.tools,
    SHA256: digest(readFileSync(role.filePath)),
  };
});
const context = loader.getAgentsFiles().agentsFiles.map(file => ({
  path: file.path, SHA256: digest(file.content),
}));
const evidence = {
  packages, cwd, agentDir, extensions, roles, context,
  skills: [...skillNames].sort(), diagnostics: skills.diagnostics,
  boundary: 'Installed discovery and registration only; no child, model or web tool execution.',
};
writeFileSync(resolve(outputArg), JSON.stringify(evidence, null, 2) + '\n');
console.log(JSON.stringify({ roles: roles.length, skills: skillNames.size, extensions: extensions.length }));
// Extensions may retain timers. All synchronous checks and the evidence write are complete.
process.exit(0);
