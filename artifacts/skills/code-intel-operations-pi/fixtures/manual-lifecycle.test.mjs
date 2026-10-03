// C-class independent example models/checklist consistency only.
// Not shipped enforcement, a config writer, process settlement or Pi qualification.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createHash } from 'node:crypto';

const reference = name => readFileSync(new URL(`../references/${name}`, import.meta.url), 'utf8');

// Each assertion group names and verifies the actual delivered checklist anchors.
function check(steps, name, body) {
  test(`${steps.join('/')}: ${name}`, () => {
    const books = { RO: 'role-overrides.md', WB: 'writer-bootstrap.md', TD: 'teardown.md', FM: 'failure-matrix.md', QUAL: 'qualification.md' };
    for (const step of steps) {
      assert.ok(reference(books[step.split('-')[0]]).includes(`## ${step} —`), step);
    }
    body();
  });
}

check(['RO-1', 'RO-2', 'WB-2', 'WB-3'], 'delivered examples are inert and explicit', () => {
  const roles = JSON.parse(reference('role-agentOverrides.example.json'));
  const privateMCP = JSON.parse(reference('private-mcp.example.json'));
  const ownership = JSON.parse(reference('private-ownership.example.json'));
  assert.ok(Object.keys(roles.subagents.agentOverrides).length > 0);
  for (const override of Object.values(roles.subagents.agentOverrides)) {
    assert.ok(override.tools.includes('read'));
    assert.ok(override.tools.includes('bash'));
    assert.ok(override.tools.includes('mcp'), 'selected pin needs the gateway, not registered direct names');
    assert.ok(override.tools.every(tool => !tool.startsWith('mcp:') && !tool.startsWith('<discovered-')));
    assert.ok(override.extensions.length > 0, 'retain provider/runtime loading');
    assert.ok(override.subagentOnlyExtensions.length > 0, 'load MCP provider explicitly');
    assert.equal(new Set(override.skills).size, override.skills.length);
  }
  const [server] = Object.values(privateMCP.mcpServers);
  assert.equal(server.directTools, false, 'proxy route must not promise direct registration');
  assert.ok(server.includeTools.length > 0, 'an empty include list allows all tools at this pin');
  assert.ok(server.includeTools.every(tool => tool.length > 0 && !/[*?]/.test(tool)));
  assert.equal(server.cwd, '/ABSOLUTE/APPROVED/PROJECT');
  assert.ok(server.args.includes(server.cwd));
  assert.ok(server.command.startsWith('/ABSOLUTE/APPROVED/'));
  assert.equal(ownership.status, 'unverified');
  assert.equal(ownership.grant.sha256, '<reviewed-grant-sha256>');
  assert.equal(ownership.settlement.state, 'unknown');
  assert.deepEqual(ownership.managedFields[0].managedValue, server);
  assert.equal(ownership.originalFile.present, false);
  assert.equal(ownership.originalFile.bytesBase64, null);
  assert.equal(ownership.approvals.piProjectResources, 'unverified');
  assert.equal(ownership.approvals.exactMCPServer, 'unverified');
  for (const name of ['role-agentOverrides.example.json', 'private-mcp.example.json', 'private-ownership.example.json']) {
    assert.doesNotMatch(reference(name), /\/home\/|\/Users\/|npx|uvx/);
  }
});

// Bounded C example assertions, not an upstream resolver or policy interceptor.
function proxyExampleConsistent(tools, server) {
  return tools.includes('mcp') && !tools.some(tool => tool.startsWith('mcp:') || tool.startsWith('<discovered-')) &&
    server.directTools === false && Array.isArray(server.includeTools) && server.includeTools.length > 0 &&
    server.includeTools.every(tool => typeof tool === 'string' && tool.length > 0 && !/[*?]/.test(tool));
}
for (const [name, tools, includeTools, directTools, expected] of [
  ['invented exact proxy surface', ['read', 'mcp'], ['fixture_lookup'], false, true],
  ['raw registered names do not select gateway', ['read', 'fixture_lookup'], ['fixture_lookup'], false, false],
  ['mcp selector is not this route', ['read', 'mcp', 'mcp:fixture_lookup'], ['fixture_lookup'], false, false],
  ['direct registration cannot repair missing selection', ['read', 'mcp'], ['fixture_lookup'], true, false],
  ['empty include list is not restricted', ['read', 'mcp'], [], false, false],
  ['wildcard include is not an exact grant', ['read', 'mcp'], ['fixture_*'], false, false],
]) {
  check(['RO-2', 'WB-2'], name, () => {
    assert.equal(proxyExampleConsistent(tools, { includeTools, directTools }), expected);
  });
}
check(['RO-2', 'WB-4'], 'route instructions disclose pin limits and proxy authority', () => {
  const doc = reference('role-overrides.md');
  for (const term of ['MCP_DIRECT_TOOLS=__none__', 'mcp-adapter.json', 'mcp.json', 'includeTools', 'cooperative']) assert.ok(doc.includes(term), term);
  assert.ok(reference('writer-bootstrap.md').includes('mcp'));
});

// Independent MANUAL-operation models only. Context booleans represent invented
// operator observations; they cannot observe real grants, races, runs or processes.
// All mutation is to in-memory clones. There is no production import or I/O write.
const clone = value => structuredClone(value);
const canonical = value => {
  if (Array.isArray(value)) return `[${value.map(canonical).join(',')}]`;
  if (value !== null && typeof value === 'object') {
    return `{${Object.keys(value).sort().map(key => `${JSON.stringify(key)}:${canonical(value[key])}`).join(',')}}`;
  }
  return JSON.stringify(value);
};
const hashBytes = bytes => createHash('sha256').update(bytes).digest('hex');
const digest = value => hashBytes(canonical(value));
function readField(config, path) {
  let value = config;
  for (const key of path) {
    if (value === null || typeof value !== 'object' || !Object.hasOwn(value, key)) {
      return { present: false, value: null };
    }
    value = value[key];
  }
  return { present: true, value: clone(value) };
}
function writeField(config, path, field) {
  let parent = config;
  for (const key of path.slice(0, -1)) parent = parent[key] ??= {};
  if (field.present) parent[path.at(-1)] = clone(field.value);
  else delete parent[path.at(-1)];
}

function manualModel(state, context, action, desired) {
  const blocked = step => ({ verdict: 'blocked', step, state: clone(state) });
  const record = state.record;
  if (!record) return blocked('TD-2');
  const grant = context.grant;
  if (!grant || grant.verified !== true || grant.owner !== record.owner ||
      grant.root !== record.root || grant.runId !== record.allocation.runId ||
      grant.action !== action || grant.inputSHA256 !== digest(state.config) ||
      grant.desiredSHA256 !== digest(desired ?? null)) return blocked('RO-3');
  if (context.owner !== record.owner || context.root !== record.root ||
      context.runId !== record.allocation.runId || context.competingOwner ||
      context.higherPrecedence || !context.inventoryKnown) return blocked('RO-1');
  if (context.runs !== 'settled-successfully' || context.unknownDescendants ||
      !context.allConsumersKnown) return blocked('TD-1');
  if (!record.originalFile || (record.originalFile.present &&
      (!record.originalFile.bytesBase64 || record.originalFile.sha256 !==
       hashBytes(Buffer.from(record.originalFile.bytesBase64, 'base64'))))) return blocked('TD-2');
  if (!['prepared', 'applied'].includes(record.status) ||
      !record.managedFields.length) return blocked('TD-2');
  if (action !== 'apply' && record.status !== 'applied') return blocked('TD-2');
  for (const field of record.managedFields) {
    if (typeof field.original?.present !== 'boolean' ||
        !Object.hasOwn(field.original, 'value') ||
        digest(field.managedValue) !== field.managedSHA256) return blocked('TD-2');
    const expected = record.status === 'prepared' ? field.original : { present: true, value: field.managedValue };
    if (digest(readField(state.config, field.path)) !== digest(expected)) return blocked('TD-2');
  }
  if (action !== 'reset' && (!desired || desired.length !== record.managedFields.length ||
      !desired.every((field, i) => digest(field.path) === digest(record.managedFields[i].path)))) return blocked('RO-2');
  if (action !== 'reset' && !context.requiredSkills.every(skill => context.effectiveSkills.includes(skill))) return blocked('RO-2');
  const next = clone(state);
  for (const [i, field] of next.record.managedFields.entries()) {
    if (action === 'reset') writeField(next.config, field.path, field.original);
    else {
      field.managedValue = clone(desired[i].value);
      field.managedSHA256 = digest(field.managedValue);
      writeField(next.config, field.path, { present: true, value: field.managedValue });
    }
  }
  next.record.history.push({ action, before: digest(state.config), after: digest(next.config) });
  next.record.status = action === 'reset' ? 'restored-pending-reload' : 'applied';
  next.record.reload = { acknowledged: false, effectivePlanEvidence: null };
  return { verdict: 'changed-pending-reload', state: next };
}

function fixture(baseline = 'absent') {
  const config = {
    unrelated: { theme: 'invented-dark' },
    subagents: { agentOverrides: { 'fixture-author': { untouched: 'keep-me' }, 'fixture-neighbor': { tools: ['read'] } } },
    mcpServers: { 'fixture-neighbor': { url: 'http://fixture.invalid/mcp' } }
  };
  const desired = [
    { path: ['subagents', 'agentOverrides', 'fixture-author', 'skills'], value: ['fixture-base', 'fixture-query'] },
    { path: ['subagents', 'agentOverrides', 'fixture-author', 'tools'], value: ['read', 'fixture-symbol'] }
  ];
  if (baseline !== 'absent') {
    for (const field of desired) writeField(config, field.path, { present: true, value: baseline === 'null' ? null : ['fixture-prior'] });
  }
  const record = JSON.parse(reference('private-ownership.example.json'));
  Object.assign(record, { owner: 'fixture-owner', root: '/ABSOLUTE/APPROVED/FIXTURE', status: 'prepared' });
  record.allocation.runId = 'fixture-run';
  const bytes = JSON.stringify(config, null, 2);
  record.originalFile = { present: true, bytesBase64: Buffer.from(bytes).toString('base64'), sha256: hashBytes(bytes) };
  record.managedFields = desired.map(field => ({ path: field.path, original: readField(config, field.path), managedValue: field.value, managedSHA256: digest(field.value) }));
  return { state: { config, record }, desired };
}
function contextFor(state, action, desired) {
  return {
    owner: 'fixture-owner', root: '/ABSOLUTE/APPROVED/FIXTURE', runId: 'fixture-run',
    inventoryKnown: true, competingOwner: false, higherPrecedence: false,
    runs: 'settled-successfully', unknownDescendants: false, allConsumersKnown: true,
    requiredSkills: ['fixture-base', 'fixture-query'], effectiveSkills: ['fixture-base', 'fixture-query'],
    grant: { verified: true, owner: 'fixture-owner', root: '/ABSOLUTE/APPROVED/FIXTURE', runId: 'fixture-run', action, inputSHA256: digest(state.config), desiredSHA256: digest(desired ?? null) }
  };
}
function applyFixture(baseline) {
  const { state, desired } = fixture(baseline);
  const result = manualModel(state, contextFor(state, 'apply', desired), 'apply', desired);
  assert.equal(result.verdict, 'changed-pending-reload');
  return { state: result.state, original: state, desired };
}
function expectBlocked(state, context, action, desired, step) {
  const before = clone(state);
  const result = manualModel(state, context, action, desired);
  assert.equal(result.verdict, 'blocked');
  assert.equal(result.step, step);
  assert.deepEqual(result.state, before, 'whole operation must preserve config and evidence');
  assert.deepEqual(state, before, 'input itself must not be mutated');
}

for (const baseline of ['absent', 'null', 'present']) {
  check(['RO-3', 'RO-5', 'TD-2', 'TD-3', 'TD-4'], `restore ${baseline} baseline without losing unrelated entries`, () => {
    const { state, original } = applyFixture(baseline);
    state.config.mcpServers['fixture-added-later'] = { url: 'http://later.invalid/mcp' };
    const expected = clone(original.config);
    expected.mcpServers['fixture-added-later'] = clone(state.config.mcpServers['fixture-added-later']);
    const result = manualModel(state, contextFor(state, 'reset'), 'reset');
    assert.equal(result.verdict, 'changed-pending-reload');
    assert.deepEqual(result.state.config, expected);
    assert.deepEqual(result.state.record.originalFile, original.record.originalFile);
    assert.equal(result.state.record.reload.acknowledged, false, 'disk restoration is not reload');
  });
}

check(['RO-3', 'RO-4', 'RO-5', 'TD-3'], 'same owner update carries first baseline through reload and restores it', () => {
  const { state, original, desired } = applyFixture('present');
  state.record.reload = { acknowledged: true, effectivePlanEvidence: 'invented-observation' };
  desired[1].value.push('fixture-callers');
  const updated = manualModel(state, contextFor(state, 'update', desired), 'update', desired);
  assert.equal(updated.verdict, 'changed-pending-reload');
  assert.deepEqual(updated.state.record.managedFields.map(f => f.original), original.record.managedFields.map(f => f.original));
  assert.equal(updated.state.record.reload.acknowledged, false);
  assert.equal(updated.state.record.history.length, 2);
  const restored = manualModel(updated.state, contextFor(updated.state, 'reset'), 'reset');
  assert.deepEqual(restored.state.config, original.config);
  assert.deepEqual(restored.state.record.originalFile, original.record.originalFile);
});

const refusalCases = [
  ['missing record', 'TD-2', (s, c) => { s.record = null; }],
  ['missing grant', 'RO-3', (s, c) => { c.grant = null; }],
  ['unverified grant', 'RO-3', (s, c) => { c.grant.verified = false; }],
  ['grant wrong root', 'RO-3', (s, c) => { c.grant.root = '/different'; }],
  ['grant wrong run', 'RO-3', (s, c) => { c.grant.runId = 'other-run'; }],
  ['grant wrong action', 'RO-3', (s, c) => { c.grant.action = 'unapproved'; }],
  ['grant stale bytes', 'RO-3', (s, c) => { c.grant.inputSHA256 = 'stale'; }],
  ['competing owner', 'RO-1', (s, c) => { c.competingOwner = true; }],
  ['different owner', 'RO-1', (s, c) => { c.owner = 'other-owner'; }],
  ['higher precedence even if equal', 'RO-1', (s, c) => { c.higherPrecedence = true; }],
  ['unknown inventory', 'RO-1', (s, c) => { c.inventoryKnown = false; }],
  ['active run', 'TD-1', (s, c) => { c.runs = 'active'; }],
  ['unknown run', 'TD-1', (s, c) => { c.runs = 'unknown'; }],
  ['failed run', 'TD-1', (s, c) => { c.runs = 'failed'; }],
  ['unknown descendant', 'TD-1', (s, c) => { c.unknownDescendants = true; }],
  ['global client unaccounted', 'TD-1', (s, c) => { c.allConsumersKnown = false; }],
  ['missing original bytes', 'TD-2', (s, c) => { s.record.originalFile.bytesBase64 = null; }],
  ['corrupt original bytes', 'TD-2', (s, c) => { s.record.originalFile.sha256 = 'corrupt'; }],
  ['missing original baseline', 'TD-2', (s, c) => { delete s.record.managedFields[0].original; }],
  ['managed record digest drift', 'TD-2', (s, c) => { s.record.managedFields[0].managedSHA256 = 'changed'; }]
];
for (const [name, step, change] of refusalCases) {
  check(['RO-1', 'RO-3', 'TD-1', 'TD-2'], `${name} blocks update and reset without changes`, () => {
    for (const action of ['update', 'reset']) {
      const { state, desired } = applyFixture('absent');
      const plan = action === 'update' ? desired : undefined;
      const context = contextFor(state, action, plan);
      change(state, context);
      expectBlocked(state, context, action, plan, step);
    }
  });
}

check(['RO-3', 'TD-2', 'TD-3'], 'one drifted field prevents all restoration; evidence is retained', () => {
  const { state } = applyFixture('null');
  const last = state.record.managedFields.at(-1);
  writeField(state.config, last.path, { present: true, value: ['operator-edit'] });
  expectBlocked(state, contextFor(state, 'reset'), 'reset', undefined, 'TD-2');
});

check(['RO-2', 'RO-4'], 'call-level skill replacement losing any mandatory skill blocks launch plan', () => {
  const { state, desired } = fixture();
  for (const missing of ['fixture-base', 'fixture-query']) {
    const context = contextFor(state, 'apply', desired);
    context.effectiveSkills = context.effectiveSkills.filter(skill => skill !== missing);
    expectBlocked(state, context, 'apply', desired, 'RO-2');
  }
});

check(['RO-3', 'TD-2'], 'initial apply refuses changed original values before any write', () => {
  const { state, desired } = fixture();
  writeField(state.config, desired.at(-1).path, { present: true, value: ['new-owner'] });
  expectBlocked(state, contextFor(state, 'apply', desired), 'apply', desired, 'TD-2');
});

check(['WB-2', 'TD-2', 'TD-3'], 'private MCP example uses record-backed leaf restoration, not whole-file replacement', () => {
  const record = JSON.parse(reference('private-ownership.example.json'));
  const { state } = fixture();
  state.record.managedFields = clone(record.managedFields);
  state.record.managedFields[0].managedSHA256 = digest(state.record.managedFields[0].managedValue);
  const desired = state.record.managedFields.map(f => ({ path: f.path, value: f.managedValue }));
  const applied = manualModel(state, contextFor(state, 'apply', desired), 'apply', desired);
  assert.equal(applied.verdict, 'changed-pending-reload');
  const reset = manualModel(applied.state, contextFor(applied.state, 'reset'), 'reset');
  assert.equal(reset.verdict, 'changed-pending-reload');
  assert.deepEqual(reset.state.config, state.config);
});

check(['FM-3'], 'resource record starts unknown, not admitted from a concurrency counter', () => {
  const example = JSON.parse(reference('failure-matrix.md').match(/```json\n([\s\S]*?)\n```/)[1]);
  for (const field of ['freeBytes', 'availableBytes', 'cgroupHeadroomBytes', 'incrementalPeakBytes']) {
    assert.equal(example[field], null, field);
  }
  assert.equal(example.observedAt, null);
  assert.equal(example.reserveBytes, 500000000);
  assert.equal(example.minimumFreeBytes, 750 * 1024 * 1024);
  assert.equal(example.minimumAvailableBytes, 750 * 1024 * 1024);
  assert.equal(example.result, 'unverified');
});

// Invented measurements only: this never reads host RAM/cgroups or starts work.
function resourceFixture() {
  const record = JSON.parse(reference('failure-matrix.md').match(/```json\n([\s\S]*?)\n```/)[1]);
  return Object.assign(record, {
    observedAt: 'invented-fresh-observation', freeBytes: 1200000000,
    availableBytes: 1600000000, cgroupHeadroomBytes: 1000000000,
    outstandingReservedBytes: 100000000, incrementalPeakBytes: 200000000,
    activeSpawns: 0, totalSpawns: 0, activeWorkers: 0, requestedWorkers: 1,
    nowMs: 1000, deadlineMs: 2000, requestedDurationMs: 1000
  });
}
check(['FM-3'], 'fresh complete budgets allow the modeled action, not a real launch', () => {
  const record = resourceFixture();
  assert.deepEqual(resourceModel(record), { expensive: 'admitted', sourceReads: 'permitted' });
  record.freeBytes = record.minimumFreeBytes;
  record.availableBytes = record.minimumAvailableBytes;
  record.outstandingReservedBytes = 0;
  record.incrementalPeakBytes = record.freeBytes - record.reserveBytes;
  assert.equal(resourceModel(record).expensive, 'admitted', 'inclusive threshold and reserve boundary');
});

// Independent arithmetic model of FM-3; granted source reads remain available.
// Freshness and authority are invented observations, never OS enforcement.
function resourceModel(record, { fresh = true, granted = true, graphWork = false, coordinator = false } = {}) {
  const blocked = { expensive: 'blocked', sourceReads: 'permitted' };
  const numbers = ['freeBytes', 'availableBytes', 'cgroupHeadroomBytes', 'minimumFreeBytes',
    'minimumAvailableBytes', 'reserveBytes', 'outstandingReservedBytes', 'incrementalPeakBytes',
    'activeSpawns', 'maxConcurrentSpawns', 'totalSpawns', 'maxTotalSpawns', 'activeWorkers',
    'requestedWorkers', 'maxWorkers', 'nowMs', 'deadlineMs', 'requestedDurationMs'];
  if (!granted || !fresh || !record.observedAt ||
      !numbers.every(key => Number.isSafeInteger(record[key]) && record[key] >= 0)) return blocked;
  if (record.freeBytes < record.minimumFreeBytes || record.availableBytes < record.minimumAvailableBytes ||
      Math.min(record.freeBytes, record.availableBytes, record.cgroupHeadroomBytes) -
        record.outstandingReservedBytes - record.incrementalPeakBytes < record.reserveBytes) return blocked;
  if (record.activeSpawns + 1 > record.maxConcurrentSpawns || record.totalSpawns + 1 > record.maxTotalSpawns ||
      record.requestedWorkers < 1 || record.activeWorkers + record.requestedWorkers > record.maxWorkers ||
      record.requestedDurationMs < 1 || record.nowMs + record.requestedDurationMs > record.deadlineMs) return blocked;
  if (graphWork && (!coordinator || !record.separatelyAdmittedCoordinatorGraphWork)) return blocked;
  return { expensive: 'admitted', sourceReads: 'permitted' };
}

for (const field of ['freeBytes', 'availableBytes', 'cgroupHeadroomBytes', 'incrementalPeakBytes', 'outstandingReservedBytes']) {
  check(['FM-3'], `unknown ${field} blocks expensive work but permits source reads`, () => {
    const record = resourceFixture();
    record[field] = null;
    const before = clone(record);
    assert.deepEqual(resourceModel(record), { expensive: 'blocked', sourceReads: 'permitted' });
    assert.deepEqual(record, before);
  });
}
for (const [name, changes] of [
  ['low free', { freeBytes: 750 * 1024 * 1024 - 1 }],
  ['low available', { availableBytes: 750 * 1024 * 1024 - 1 }],
  ['low cgroup headroom despite idle counter', { cgroupHeadroomBytes: 799999999 }],
  ['pending peaks consume reserve despite idle counter', { outstandingReservedBytes: 300000001 }],
  ['concurrent limit', { activeSpawns: 1 }],
  ['total limit despite idle concurrency', { totalSpawns: 4 }],
  ['worker limit', { activeWorkers: 1 }],
  ['deadline', { requestedDurationMs: 1001 }],
  ['unknown deadline', { deadlineMs: null }],
  ['unknown worker request', { requestedWorkers: null }],
  ['unknown concurrent limit', { maxConcurrentSpawns: null }],
  ['unknown total limit', { maxTotalSpawns: null }],
  ['missing observation', { observedAt: null }]
]) {
  check(['FM-3'], `${name} independently blocks expensive work, not bounded source reads`, () => {
    assert.deepEqual(resourceModel(Object.assign(resourceFixture(), changes)), { expensive: 'blocked', sourceReads: 'permitted' });
  });
}
check(['FM-3'], 'stale measurements, absent grant and child graph subprocess work are not admitted', () => {
  const record = resourceFixture();
  for (const options of [{ fresh: false }, { granted: false }, { graphWork: true }, { graphWork: true, coordinator: true }]) {
    assert.equal(resourceModel(record, options).expensive, 'blocked');
  }
  record.separatelyAdmittedCoordinatorGraphWork = true;
  assert.equal(resourceModel(record, { graphWork: true }).expensive, 'blocked', 'grant never turns a child into coordinator');
  assert.equal(resourceModel(record, { graphWork: true, coordinator: true }).expensive, 'admitted');
  record.freeBytes = 1;
  assert.equal(resourceModel(record, { graphWork: true, coordinator: true }).expensive, 'blocked', 'graph grant cannot waive RAM');
});

check(['FM-4', 'TD-2', 'TD-3'], 'interrupted reset retains partial result and baseline; blind replay is blocked', () => {
  const { state } = applyFixture('present');
  const first = state.record.managedFields[0];
  const baseline = clone(state.record.originalFile);
  const beforeHash = digest(state.config);
  writeField(state.config, first.path, first.original);
  state.record.history.push({ action: 'reset-interrupted', before: beforeHash, after: digest(state.config), completedPaths: [first.path] });
  expectBlocked(state, contextFor(state, 'reset'), 'reset', undefined, 'TD-2');
  assert.deepEqual(state.record.originalFile, baseline);
  assert.equal(state.record.history.at(-1).action, 'reset-interrupted');
});

check(['FM-1', 'FM-2', 'QUAL-2', 'QUAL-4'], 'failure checklist and Q handoff retain distinct denials and evidence classes', () => {
  const failures = reference('failure-matrix.md');
  for (const term of ['one attempted readiness check', 'expected **and observed**',
    'maximum additional attempts', 'Missing endpoint', 'Missing provider', 'Missing/broken LSP',
    'Wrong root', 'Stale alias', 'Missing/malformed graph', 'Historical graph',
    'Denied Pi project trust', 'Denied/absent MCP approval', 'Version/schema mismatch']) {
    assert.ok(failures.includes(term), term);
  }
  const qualification = reference('qualification.md');
  for (let i = 1; i <= 7; i++) assert.ok(qualification.includes(`| I-T${i},`), `I-T${i} handoff row`);
  for (const term of ['integration-only', 'C example models here cannot fill M enforcement or I outcomes',
    'OP-TRUST-MCP', 'OP-APPROVAL-MCP', 'Sentinel tests alone never prove service survival']) {
    assert.ok(qualification.includes(term), term);
  }
});
