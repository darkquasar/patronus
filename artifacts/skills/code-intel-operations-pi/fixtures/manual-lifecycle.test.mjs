// C-class distributed-content consistency only.
// This fixture never installs packages, imports Pi extensions, starts services,
// reads host resources or qualifies runtime behavior.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const reference = name => readFileSync(new URL(`../references/${name}`, import.meta.url), 'utf8');
const json = name => JSON.parse(reference(name));

function hasHeadings(file, prefix, count) {
  const text = reference(file);
  for (let i = 1; i <= count; i++) {
    assert.match(text, new RegExp(`## ${prefix}-${i}(?:[: —])`), `${file}: ${prefix}-${i}`);
  }
}

test('shared HTTP example is gateway-only and exact', () => {
  const config = json('shared-mcp.example.json');
  assert.deepEqual(Object.keys(config.mcpServers).sort(), ['graphify-shared-pi', 'serena-shared-pi']);
  assert.equal(config.mcpServers['serena-shared-pi'].url, 'http://127.0.0.1:9121/mcp');
  assert.equal(config.mcpServers['graphify-shared-pi'].url, 'http://127.0.0.1:9122/mcp');
  for (const server of Object.values(config.mcpServers)) {
    assert.equal(server.directTools, false);
    assert.equal(Object.hasOwn(server, 'command'), false, 'children must not receive a stdio launcher');
  }
  for (const key of ['allowInstall', 'sampling', 'samplingAutoApprove', 'jev', 'scriptMode', 'autoAuth', 'elicitation']) {
    assert.equal(config.settings[key], false, key);
  }
  assert.equal(config.settings.hostConfigDiscovery, 'off');
  assert.deepEqual(config.settings.ancestorConfigRoots, []);
});

test('service descriptor binds one coordinator-owned pair and source-backed argv', () => {
  const descriptor = json('service-descriptor.example.json');
  const contract = json('runtime-command-contract.json');
  assert.equal(descriptor.owner, '<coordinator-run-identity>');
  assert.equal(descriptor.services.length, 2);
  const byName = Object.fromEntries(descriptor.services.map(service => [service.entryName, service]));
  assert.equal(byName['serena-shared-pi'].pin, '7a2968335f2198b966864de1ce3655c8e485a653');
  assert.equal(byName['graphify-shared-pi'].pin, 'graphifyy[mcp]==0.9.31');
  assert.deepEqual(byName['serena-shared-pi'].argv, contract.serena.argv);
  assert.deepEqual(byName['graphify-shared-pi'].argv, contract.graphify.argv);
  assert.equal(contract.runtimeExecuted, false);
  assert.equal(contract.serena.sourceRevision, '7a2968335f2198b966864de1ce3655c8e485a653');
  assert.equal(contract.graphify.sourceRevision, '4fe11092ccbe9f543608f140c790f68d5d83cae4');
  assert.deepEqual(contract.graphify.observedContract.consoleScripts, ['graphify', 'graphify-mcp']);
  assert.equal(descriptor.readiness.status, 'runtime-unverified');
});

test('lifecycle checklists retain every numbered gate', () => {
  hasHeadings('shared-services.md', 'SS', 6);
  hasHeadings('role-overrides.md', 'RO', 5);
  hasHeadings('writer-bootstrap.md', 'WB', 4);
  hasHeadings('teardown.md', 'TD', 4);
  hasHeadings('failure-matrix.md', 'FM', 4);
  hasHeadings('qualification.md', 'QUAL', 4);
});

test('shared-service runbook uses delivered executables and loopback only', () => {
  const runbook = reference('shared-services.md');
  for (const term of [
    'pi-mcp-adapter@3.0.0',
    '7a2968335f2198b966864de1ce3655c8e485a653',
    'graphifyy[mcp]==0.9.31',
    '--transport streamable-http',
    '--host 127.0.0.1',
    '--port 9121',
    '--transport http',
    '--port 9122',
    '--stateless'
  ]) assert.ok(runbook.includes(term), term);
  assert.match(runbook, /refuse\s+startup when either endpoint already has an unknown listener/i);
  assert.match(runbook, /Children never build or refresh the graph/i);
  assert.match(runbook, /protected digest-named\s+copy/i);
  assert.match(runbook, /launch core roles with `async: true`/i);
  assert.doesNotMatch(runbook, /graphify-mcp \/ABSOLUTE\/APPROVED\/PROJECT\/graphify-out\/graph\.json/);
});

test('role guidance fails closed for foreground and private service paths', () => {
  const roles = reference('role-overrides.md');
  const readiness = reference('readiness.md');
  assert.match(roles, /must run as\s+background children with `async: true`/i);
  assert.match(roles, /foreground child[\s\S]*fails before its first model turn/i);
  assert.match(readiness, /reject stdio\/command\s+Serena or Graphify entries/i);
  assert.match(readiness, /never mutable\s+`graphify-out\/graph\.json`/i);
});

test('writer guidance forbids private service fallback', () => {
  const writer = reference('writer-bootstrap.md');
  assert.match(writer, /does not bootstrap a private MCP process/i);
  for (const term of ['never starts `serena`', '`graphify-mcp`', '`uvx`', 'coordinator-owned HTTP entries']) {
    assert.ok(writer.includes(term), term);
  }
});

test('resource example starts unknown and keeps conservative floors', () => {
  const failures = reference('failure-matrix.md');
  const record = JSON.parse(failures.match(/```json\n([\s\S]*?)\n```/)[1]);
  for (const field of ['freeBytes', 'availableBytes', 'cgroupHeadroomBytes', 'incrementalPeakBytes']) {
    assert.equal(record[field], null, field);
  }
  assert.equal(record.reserveBytes, 500000000);
  assert.equal(record.minimumFreeBytes, 750 * 1024 * 1024);
  assert.equal(record.minimumAvailableBytes, 750 * 1024 * 1024);
  assert.equal(record.result, 'unverified');
});

test('qualification keeps static placement separate from runtime proof', () => {
  const qualification = reference('qualification.md');
  for (let i = 1; i <= 7; i++) assert.ok(qualification.includes(`| I-T${i},`), `I-T${i}`);
  for (const term of [
    'placed,\nruntime-unverified',
    'Sentinel tests alone never prove service survival',
    'pi-mcp-adapter-3.0.0.json',
    'serena-7a296833.json',
    'graphifyy-0.9.31.json'
  ]) assert.ok(qualification.includes(term), term);
});
