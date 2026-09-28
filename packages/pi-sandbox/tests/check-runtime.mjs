// Run inside a fresh sandbox; checks package loading without a model request.
import assert from 'node:assert/strict';
import { execFileSync, spawn } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

const globalRoot = execFileSync('npm', ['root', '-g'], { encoding: 'utf8' }).trim();
const piRoot = join(globalRoot, '@earendil-works/pi-coding-agent');
const agentDir = join(homedir(), '.pi/agent');
const extensionRoot = join(agentDir, 'npm/node_modules/pi-subagents');
const readJSON = path => JSON.parse(readFileSync(path, 'utf8'));
assert.equal(readJSON(join(piRoot, 'package.json')).version, '0.87.1');
assert.equal(readJSON(join(extensionRoot, 'package.json')).version, '0.71.0');
assert.ok(readJSON(join(agentDir, 'settings.json')).packages.includes('npm:pi-subagents@0.71.0'));

const { DefaultResourceLoader } = await import(pathToFileURL(join(piRoot, 'dist/index.js')).href);
const loader = new DefaultResourceLoader({ cwd: '/tmp', agentDir });
await loader.reload();
const result = loader.getExtensions();
assert.deepEqual(result.errors, [], 'Extensions must load without peer/compat errors');
const extension = result.extensions.find(item => item.resolvedPath.startsWith(extensionRoot + '/'));
assert.ok(extension, 'Native Pi package discovery must load pi-subagents');
assert.ok(extension.tools.has('subagent'), 'Delegation tool must register');
assert.ok(extension.commands.has('subagents-doctor'), 'Diagnostics command must register');
console.log('PASS Pi 0.87.1 + pi-subagents 0.71.0: package discovery, extension loading, delegation tool and diagnostics');

// Exercise the real CLI's extension/session initialization as well as the SDK.
await new Promise((resolve, reject) => {
  const child = spawn('pi', ['--mode', 'rpc', '--no-session'], { cwd: '/tmp' });
  let output = '';
  let errors = '';
  let ready = false;
  const timer = setTimeout(() => {
    child.kill('SIGKILL');
    reject(new Error('Native Pi RPC startup timed out'));
  }, 30000);
  child.stderr.on('data', chunk => { errors += chunk; });
  child.stdout.on('data', chunk => {
    output += chunk;
    let newline;
    while ((newline = output.indexOf('\n')) >= 0) {
      const line = output.slice(0, newline);
      output = output.slice(newline + 1);
      let message;
      try { message = JSON.parse(line); } catch { continue; }
      if (message.id !== 'kit-smoke') continue;
      ready = message.success === true && message.data?.commands?.some(
        command => command.name === 'subagents-doctor' && command.source === 'extension');
      child.stdin.end();
    }
  });
  child.on('error', error => { clearTimeout(timer); reject(error); });
  child.on('close', code => {
    clearTimeout(timer);
    if (code !== 0 || !ready || /Failed to load extension|Could not .*Pi installation/.test(errors)) {
      reject(new Error(`Native Pi extension startup failed: exit=${code}, registered=${ready}, ${errors}`));
    } else resolve();
  });
  child.stdin.write(JSON.stringify({ id: 'kit-smoke', type: 'get_commands' }) + '\n');
});
console.log('PASS native Pi RPC startup: subagents diagnostics registered, no model request');
