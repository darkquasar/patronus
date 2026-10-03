import test from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { mkdtempSync, readFileSync, writeFileSync, rmSync, unlinkSync, symlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const require = createRequire(import.meta.url);
const { evaluateStage, workflowExample, sha256 } = require('../references/workflow-example.js');
const schema = require('../references/stage-gate.schema.json');

// Validate every keyword used in the delivered schema, not a second gate policy.
// Deliberately not a general JSON Schema implementation or runtime qualification.
function validate(value, rule) {
  const supported = new Set(['$schema', 'title', 'type', 'enum', 'anyOf', 'required', 'properties', 'additionalProperties', 'items', 'minLength', 'pattern']);
  for (const key of Object.keys(rule)) assert.ok(supported.has(key), `unsupported schema keyword ${key}`);
  if (rule.anyOf) { assert.ok(rule.anyOf.some(r => { try { validate(value, r); return true; } catch { return false; } })); return; }
  if (rule.enum) assert.ok(rule.enum.includes(value));
  if (rule.type === 'null') assert.equal(value, null);
  if (rule.type === 'string') {
    assert.equal(typeof value, 'string');
    if (rule.minLength) assert.ok(value.length >= rule.minLength);
    if (rule.pattern) assert.match(value, new RegExp(rule.pattern));
  }
  if (rule.type === 'boolean') assert.equal(typeof value, 'boolean');
  if (rule.type === 'array') { assert.ok(Array.isArray(value)); value.forEach(v => validate(v, rule.items)); }
  if (rule.type === 'object') {
    assert.ok(value && typeof value === 'object' && !Array.isArray(value));
    rule.required.forEach(k => assert.ok(Object.hasOwn(value, k), `missing ${k}`));
    for (const [k, v] of Object.entries(value)) {
      if (rule.additionalProperties === false) assert.ok(Object.hasOwn(rule.properties, k), `extra ${k}`);
      validate(v, rule.properties[k]);
    }
  }
}
function fixture(t) {
  const root = mkdtempSync(join(tmpdir(), 'pi-workflow-example-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const input = Buffer.from('invented requirements, not a catalog identity\n');
  writeFileSync(join(root, 'brief.txt'), input);
  const report = { criteriaSatisfied: [{ id: 'behavior', status: 'satisfied', evidence: 'observed invented check' }], changedFiles: [], residualRisks: [] };
  writeFileSync(join(root, 'ignored-report.json'), JSON.stringify(report));
  const outputHash = sha256(readFileSync(join(root, 'ignored-report.json')));
  const check = { passed: true, cached: false, invocationId: 'fresh-host-check-1', mode: 'json', sha256: outputHash };
  const record = {
    sourceRoot: root, expectedSourceRoot: root, revision: 'unchanged-invented-head', briefSha256: sha256(input),
    authority: { ownerVerified: true, issuer: 'fixture-owner', issuedAt: '2026-10-01', revalidationTrigger: 'changed scope/bytes',
      revalidationRequired: false, expired: false, sourceRoot: root, briefSha256: sha256(input), stage: 'planning', action: 'close', outputs: ['next-plan.md'] },
    request: { nextStage: 'planning', action: 'close', outputs: ['next-plan.md'], runs: 0, reviewCycles: 0, expensive: false },
    runs: [{ key: 'question', runId: 'run-1', sessionId: 'session-1', missionId: 'mission-1', attempt: 1, owner: 'leaf', settled: true, ok: true }],
    inputs: [{ path: 'brief.txt', sha256: sha256(input) }],
    outputs: [{ path: 'ignored-report.json', sha256: outputHash, verification: check }],
    requiredRuns: ['question'], requiredOutputs: ['ignored-report.json'], requiredCriteria: ['behavior'], reviews: [{ path: 'ignored-report.json', sha256: outputHash, reviewer: 'fresh-reviewer', independent: true, settled: true }],
    corrections: [], findings: [], gates: { requirements: 'passed', tests: 'passed', authority: 'passed', release: 'passed' },
    budget: { reviewCycles: 1, maxReviewCycles: 2, runsUsed: 3, maxRuns: 3, maxConcurrent: 1, elapsedMs: 10, maxElapsedMs: 100 },
    memory: { freeMiB: 1200, availableMiB: 1300, containerHeadroomMiB: 1100, estimatedMiB: 200 }
  };
  return { root, record, report };
}
function verdict(record, expected, reason) {
  const result = evaluateStage(record);
  validate(result, schema);
  assert.equal(result.verdict, expected, JSON.stringify(result));
  if (reason) assert.ok(result.reasons.includes(reason), JSON.stringify(result.reasons));
  return result;
}
function finding(id, severity = 'Medium') {
  return { id, sources: [{ id: `${id}-source`, originalSeverity: severity, consequenceEvidence: '' }], severity,
    rationale: 'bounded material consequence supported by source', evidence: 'invented source observation', location: 'fixture:1', unresolved: true,
    residual: { owner: 'fixture-owner', reason: 'bounded consequence accepted', scope: 'fixture planning only' } };
}

test('missing evidence blocks rather than claiming a stage transition', () => {
  verdict({}, 'blocked', 'source');
});
for (const n of [0, 1, 2]) test(`${n} accepted Medium defects close after one cycle without spare runs`, t => {
  const { record } = fixture(t);
  record.findings = Array.from({ length: n }, (_, i) => finding(`defect-${i}`));
  verdict(record, 'ready');
});
test('three distinct Medium block', t => {
  const { record } = fixture(t);
  record.findings = [finding('a'), finding('b'), finding('c')];
  verdict(record, 'blocked', 'reviewDisposition');
});
test('duplicate Medium counts once and preserves all original source IDs', t => {
  const { record } = fixture(t);
  const duplicate = finding('a'); duplicate.sources[0].id = 'other-reviewer-a';
  record.findings = [finding('a'), duplicate, finding('b')];
  const result = verdict(record, 'ready');
  assert.equal(result.findings.length, 2);
  assert.deepEqual(result.findings[0].sources.map(s => s.id), ['a-source', 'other-reviewer-a']);
});
for (const severity of ['Critical', 'Major']) test(`${severity} blocks even with residual owner acceptance`, t => {
  const { record } = fixture(t); record.findings = [finding('a', severity)];
  verdict(record, 'blocked', 'reviewDisposition');
});
test('Low-only correction is optional', t => {
  const { record } = fixture(t); record.findings = [finding('a', 'Low')]; record.findings[0].residual = null;
  verdict(record, 'ready');
});
test('Medium needs explicit owner residual acceptance', t => {
  const { record } = fixture(t); record.findings = [finding('a')]; record.findings[0].residual = null;
  verdict(record, 'blocked', 'reviewDisposition');
});
test('severity downgrade and legacy conversion require consequence evidence', t => {
  const { record } = fixture(t); record.findings = [finding('a')];
  record.findings[0].sources[0].originalSeverity = 'Major';
  verdict(record, 'blocked', 'reviewDisposition');
  record.findings[0].sources[0].originalSeverity = 'Important';
  verdict(record, 'blocked', 'reviewDisposition');
  record.findings[0].sources[0].consequenceEvidence = 'verified bounded consequence, parent disposition, not threshold gaming';
  verdict(record, 'ready');
});
for (const field of ['requirements', 'tests', 'authority', 'release']) test(`independent ${field} failure blocks with zero findings`, t => {
  const { record } = fixture(t); record.gates[field] = 'failed';
  verdict(record, 'blocked', `${field}Gate`);
});
for (const [name, alter, reason] of [
  ['missing output', (r, root) => unlinkSync(join(root, 'ignored-report.json')), 'outputsAcceptanceAndFreshness'],
  ['wrong-stage grant', r => { r.authority.stage = 'research'; }, 'authority'],
  ['missing authority', r => { delete r.authority; }, 'authority'],
  ['expired grant', r => { r.authority.expired = true; }, 'authority'],
  ['root mismatch', r => { r.expectedSourceRoot += '-other'; }, 'source'],
  ['unsettled run', r => { r.runs[0].settled = false; }, 'runsSettled'],
  ['child dispatch receipt', r => { r.runs[0].state = 'running'; r.runs[0].dispatchReceipt = true; }, 'runsSettled'],
  ['omitted required run', r => { r.requiredRuns.push('missing'); }, 'runsSettled'],
  ['omitted required output', r => { r.requiredOutputs.push('absent.json'); }, 'outputsAcceptanceAndFreshness'],
  ['absent review', r => { r.reviews = []; }, 'reviewsForBytes'],
  ['stale review', r => { r.reviews[0].sha256 = 'a'.repeat(64); }, 'reviewsForBytes'],
  ['missing criteria', r => { r.requiredCriteria = []; }, 'outputsAcceptanceAndFreshness'],
  ['cached success beside fresh hash', r => { r.outputs[0].verification.cached = true; }, 'outputsAcceptanceAndFreshness'],
  ['unbound digest invocation', r => { r.outputs[0].verification.mode = 'digest-invocation'; r.outputs[0].verification.expectedDigest = r.outputs[0].sha256; }, 'outputsAcceptanceAndFreshness'],
  ['third completed cycle', r => { r.budget.reviewCycles = 3; }, 'budgetNotExceeded'],
  ['over run budget', r => { r.budget.runsUsed = 4; }, 'budgetNotExceeded'],
  ['over deadline', r => { r.budget.elapsedMs = 101; }, 'budgetNotExceeded'],
  ['no deadline for consuming action', r => { r.budget.elapsedMs = 100; r.request.expensive = true; }, 'actionAdmission'],
  ['over concurrency admission', r => { r.budget.maxRuns = 8; r.request.runs = 2; }, 'actionAdmission']
]) test(name, t => { const { record, root } = fixture(t); alter(record, root); verdict(record, 'blocked', reason); });

test('valid existing planning grant remains usable', t => { const { record } = fixture(t); verdict(record, 'ready'); });
test('clean final wave closes with zero remaining waves; extra wave refused', t => {
  const { record } = fixture(t); record.budget.reviewCycles = 2;
  verdict(record, 'ready'); record.request.reviewCycles = 1;
  verdict(record, 'blocked', 'actionAdmission');
});
for (const [name, memory] of [
  ['low free RAM', { freeMiB: 749 }], ['low available RAM', { availableMiB: 749 }],
  ['unknown headroom', { containerHeadroomMiB: null }], ['insufficient reserve', { containerHeadroomMiB: 699 }]
]) test(name, t => {
  const { record } = fixture(t); record.request.expensive = true; Object.assign(record.memory, memory);
  verdict(record, 'blocked', 'actionAdmission');
  record.request.expensive = false; verdict(record, 'ready'); // closing consumes no memory admission
});
test('adequate measured headroom admits an expensive consuming action with capacity', t => {
  const { record } = fixture(t); record.request.expensive = true; record.request.runs = 1; record.budget.maxRuns = 4;
  verdict(record, 'ready');
});
test('malformed acceptance cannot be rescued by a matching digest', t => {
  const { record, root } = fixture(t);
  writeFileSync(join(root, 'ignored-report.json'), '{"criteriaSatisfied":"DONE"}');
  const hash = sha256(readFileSync(join(root, 'ignored-report.json')));
  record.outputs[0].sha256 = hash; record.outputs[0].verification.sha256 = hash; record.reviews[0].sha256 = hash;
  verdict(record, 'blocked', 'outputsAcceptanceAndFreshness');
});
test('ignored bytes change with unchanged HEAD; fresh correction verification does not claim rereview', t => {
  const { record, root, report } = fixture(t); const oldHash = record.outputs[0].sha256; const head = record.revision;
  verdict(record, 'ready');
  report.residualRisks = ['changed ignored artifact']; writeFileSync(join(root, 'ignored-report.json'), JSON.stringify(report));
  verdict(record, 'blocked', 'outputsAcceptanceAndFreshness'); assert.equal(record.revision, head);
  const newHash = sha256(readFileSync(join(root, 'ignored-report.json')));
  record.outputs[0].sha256 = newHash; record.outputs[0].verification.sha256 = newHash;
  verdict(record, 'blocked', 'reviewsForBytes');
  const fresh = { passed: true, cached: false, mode: 'digest-invocation', expectedDigest: newHash, sha256: newHash, invocationId: `verify-${newHash}` };
  record.outputs[0].verification = fresh; record.budget.reviewCycles = 2;
  record.corrections = [{ path: 'ignored-report.json', oldReviewedSha256: oldHash, newSha256: newHash, disposition: 'parent verified at ceiling', independentlyReviewed: false, parentVerification: fresh }];
  const corrected = verdict(record, 'ready');
  assert.equal(corrected.corrections[0].oldReviewedSha256, oldHash);
  assert.equal(corrected.corrections[0].newSha256, newHash);
  assert.equal(corrected.corrections[0].independentlyReviewed, false);
  record.corrections[0].independentlyReviewed = true;
  verdict(record, 'blocked', 'reviewsForBytes');
});
test('unsafe evidence and symlinks are refused', t => {
  const { record, root } = fixture(t); symlinkSync('brief.txt', join(root, 'link'));
  record.inputs[0].path = 'link'; verdict(record, 'blocked', 'inputs');
  record.inputs[0].path = '../outside'; verdict(record, 'blocked', 'inputs');
});
test('conflicting canonical records and reused source IDs cannot reduce blocker count', t => {
  const { record } = fixture(t); record.findings = [finding('a'), finding('a', 'Low')];
  verdict(record, 'blocked', 'reviewDisposition');
  record.findings = [finding('a'), finding('b')]; record.findings[1].sources = record.findings[0].sources;
  verdict(record, 'blocked', 'reviewDisposition');
});
test('schema rejects malformed typed verdicts', () => {
  assert.throws(() => validate({ verdict: 'ready' }, schema));
  assert.throws(() => validate({ ...evaluateStage({}), verdict: 'DONE' }, schema));
});

const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
function launchInputs(root) {
  const item = key => ({ key, label: `Inspect ${key}`, agent: 'invented-leaf', task: `Read mandatory brief/ledger. Answer ${key}. Return complete output; no delegation.`, output: join(root, `${key}.json`), gateCommand: 'invented-approved-host-check' });
  return { cwd: root, lanes: [item('slow'), item('fast')], synthesis: item('synthesis') };
}
function fakeRuns(root, record, events, blockedKey, receiptKey, throwsKey) {
  const complete = async item => {
    events.push(`start:${item.key}`);
    if (item.key === throwsKey) { events.push(`settled:${item.key}`); throw new Error('invented child failure'); }
    if (item.key === 'slow') await new Promise(resolve => setTimeout(resolve, 10));
    if (item.key === 'synthesis') {
      assert.ok(events.includes('settled:slow') && events.includes('settled:fast'));
      for (const key of ['slow', 'fast']) {
        assert.ok(item.task.includes(join(root, `${key}.json`)));
        assert.equal(JSON.parse(readFileSync(join(root, `${key}.json`), 'utf8')).answer, key);
      }
    }
    assert.equal(item.async, undefined); // completed results, not async child receipts
    assert.equal(item.context, 'fresh'); assert.equal(item.gate.output, 'json');
    assert.deepEqual(item.gate.schema, schema);
    const report = { answer: item.key, returnedCompleteContent: true, changedFiles: [], residualRisks: [],
      criteriaSatisfied: [{ id: 'behavior', status: 'satisfied', evidence: 'invented task observation' }] };
    writeFileSync(item.output, JSON.stringify(report));
    const hostRecord = structuredClone(record);
    const output = hostRecord.outputs[0];
    output.path = `${item.key}.json`; hostRecord.requiredOutputs = [output.path]; output.sha256 = sha256(readFileSync(item.output));
    output.verification.sha256 = output.sha256;
    hostRecord.reviews[0].path = output.path; hostRecord.reviews[0].sha256 = output.sha256;
    const gate = evaluateStage(hostRecord); validate(gate, item.gate.schema);
    if (item.key === blockedKey) { gate.verdict = 'blocked'; gate.reasons.push('invented failure'); }
    events.push(`settled:${item.key}`);
    return { key: item.key, ok: item.key !== blockedKey, ...(item.key === receiptKey ? { state: 'running' } : {}), runId: `run-${item.key}`,
      output: '', outputReference: item.output, artifactPaths: [item.output], structuredOutput: gate };
  };
  return { run: async (key, item) => complete({ ...item, key }),
    // Native runs.all config items request collectFailure; rejected children are
    // returned as failed results while the other ordered items finish.
    all: async items => Promise.all(items.map(item => complete(item).catch(error => ({ key: item.key, ok: false, error: error.message })))) };
}
test('native-shape doubles preserve ordered results, saved bytes and awaited synthesis', async t => {
  const { record, root } = fixture(t); const events = []; const args = launchInputs(root);
  const result = await new AsyncFunction('runs', workflowExample(args))(fakeRuns(root, record, events));
  assert.equal(result.verdict, 'ready');
  assert.deepEqual(result.inputs, args.lanes.map(l => l.output));
  assert.ok(events.indexOf('settled:fast') < events.indexOf('settled:slow')); // completion differs from result order
  assert.equal(JSON.parse(readFileSync(args.synthesis.output, 'utf8')).answer, 'synthesis');
});
for (const mode of ['blocked', 'receipt', 'rejected']) test(`${mode} child holds transition until all independent runs settle`, async t => {
  const { record, root } = fixture(t); const events = [];
  const result = await new AsyncFunction('runs', workflowExample(launchInputs(root)))(fakeRuns(root, record, events, mode === 'blocked' ? 'fast' : undefined, mode === 'receipt' ? 'fast' : undefined, mode === 'rejected' ? 'fast' : undefined));
  assert.equal(result.verdict, 'blocked');
  assert.ok(events.includes('settled:slow') && events.includes('settled:fast'));
  assert.ok(!events.includes('start:synthesis'));
});
test('duplicate runtime outputs refuse launch construction', t => {
  const { root } = fixture(t); const args = launchInputs(root); args.lanes[1].output = args.lanes[0].output;
  assert.throws(() => workflowExample(args), /duplicate key\/output/);
});
