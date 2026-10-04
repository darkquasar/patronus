'use strict';

// EXAMPLE only. Host-side checklist over invented/owner-reviewed evidence, not an
// authorization service. Never import this module inside Pi's workflow sandbox.
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const schema = require('./stage-gate.schema.json');
const sha256 = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const text = value => typeof value === 'string' && value.trim().length > 0;
const digest = value => typeof value === 'string' && /^[a-f0-9]{64}$/.test(value);
const integer = value => Number.isSafeInteger(value) && value >= 0;
const severities = ['Low', 'Medium', 'Major', 'Critical'];

function readEvidence(root, item) {
  if (!item || !text(item.path) || path.isAbsolute(item.path) ||
      item.path.split(/[\\/]/).some(part => !part || part === '.' || part === '..')) {
    throw new Error('unsafe evidence path');
  }
  let current = root;
  for (const part of item.path.split('/')) {
    current = path.join(current, part);
    if (fs.lstatSync(current).isSymbolicLink()) throw new Error('symlink evidence');
  }
  const stat = fs.statSync(current);
  if (!stat.isFile() || stat.size > 1024 * 1024) throw new Error('unreadable/big evidence');
  const bytes = fs.readFileSync(current);
  if (!digest(item.sha256) || sha256(bytes) !== item.sha256) throw new Error('changed evidence');
  return bytes;
}

function freshCheck(check, hash) {
  return check && check.passed === true && check.cached === false &&
    text(check.invocationId) && check.sha256 === hash &&
    (check.mode === 'json' || (check.mode === 'digest-invocation' &&
      check.expectedDigest === hash && check.invocationId.includes(hash)));
}

function canonicalFindings(reports) {
  if (!Array.isArray(reports)) throw new Error('missing findings');
  const canonical = new Map();
  const sourceOwners = new Map();
  for (const finding of reports) {
    if (!finding || !text(finding.id) || !text(finding.rationale) ||
        !text(finding.evidence) || !text(finding.location) ||
        !severities.includes(finding.severity) || typeof finding.unresolved !== 'boolean' ||
        !Array.isArray(finding.sources) || finding.sources.length === 0) {
      throw new Error('malformed finding or normalization rationale');
    }
    const residual = finding.residual;
    if (residual !== null && (!residual || !text(residual.owner) ||
        !text(residual.reason) || !text(residual.scope))) throw new Error('malformed residual');
    for (const source of finding.sources) {
      if (!source || !text(source.id) || !text(source.originalSeverity)) throw new Error('missing original finding');
      const original = severities.indexOf(source.originalSeverity);
      if ((original < 0 || original > severities.indexOf(finding.severity)) &&
          !text(source.consequenceEvidence)) throw new Error('unsupported severity conversion/downgrade');
      if (sourceOwners.has(source.id) && sourceOwners.get(source.id) !== finding.id) {
        throw new Error('source ID assigned to multiple defects');
      }
      sourceOwners.set(source.id, finding.id);
    }
    const prior = canonical.get(finding.id);
    if (prior) {
      for (const field of ['severity', 'unresolved', 'rationale', 'evidence', 'location', 'residual']) {
        if (JSON.stringify(prior[field]) !== JSON.stringify(finding[field])) throw new Error('conflicting duplicate defect');
      }
      for (const source of finding.sources) {
        const existing = prior.sources.find(s => s.id === source.id);
        if (existing && JSON.stringify(existing) !== JSON.stringify(source)) throw new Error('conflicting source ID');
        if (!existing) prior.sources.push({ ...source, consequenceEvidence: source.consequenceEvidence ?? '' });
      }
    } else canonical.set(finding.id, { ...finding, sources: finding.sources.map(s => ({ ...s, consequenceEvidence: s.consequenceEvidence ?? '' })) });
  }
  return [...canonical.values()];
}

function evaluateStage(record) {
  const reasons = [];
  const predicates = {};
  const evidence = [];
  const corrections = [];
  let findings = [];
  const check = (key, fn) => {
    try { predicates[key] = fn() === true; } catch { predicates[key] = false; }
    if (!predicates[key]) reasons.push(key);
  };
  // These are host/owner observations, not self-authenticating child grants.
  const request = record?.request;
  const budget = record?.budget;
  let root;
  check('source', () => {
    root = fs.realpathSync(record.sourceRoot);
    return root === record.sourceRoot && root === record.expectedSourceRoot &&
      text(record.revision) && digest(record.briefSha256);
  });
  check('authority', () => {
    const g = record.authority;
    return g && text(request?.nextStage) && text(request?.action) &&
      g.ownerVerified === true && text(g.issuer) && text(g.issuedAt) &&
      text(g.revalidationTrigger) && g.revalidationRequired === false && g.expired === false &&
      g.sourceRoot === root && g.briefSha256 === record.briefSha256 &&
      g.stage === request?.nextStage && g.action === request?.action &&
      Array.isArray(g.outputs) && Array.isArray(request.outputs) &&
      request.outputs.every(p => text(p) && g.outputs.includes(p));
  });
  check('runsSettled', () => Array.isArray(record.runs) && record.runs.length > 0 &&
    Array.isArray(record.requiredRuns) && record.requiredRuns.length > 0 &&
    record.requiredRuns.every(text) &&
    new Set(record.runs.map(r => r.key)).size === record.runs.length &&
    record.requiredRuns.every(key => record.runs.some(r => r.key === key)) &&
    record.runs.every(r => text(r.key) && text(r.runId) && text(r.sessionId) && text(r.missionId) &&
      integer(r.attempt) && text(r.owner) && r.settled === true && r.ok === true &&
      r.state !== 'running' && r.dispatchReceipt !== true));
  check('inputs', () => Array.isArray(record.inputs) && record.inputs.length > 0 &&
    record.inputs.every(item => { readEvidence(root, item); return true; }));
  check('outputsAcceptanceAndFreshness', () => {
    if (!Array.isArray(record.outputs) || !record.outputs.length ||
        !Array.isArray(record.requiredCriteria) || !record.requiredCriteria.length ||
        !record.requiredCriteria.every(text) || !Array.isArray(record.requiredOutputs) ||
        !record.requiredOutputs.length || !record.requiredOutputs.every(text) ||
        new Set(record.outputs.map(o => o.path)).size !== record.outputs.length ||
        !record.requiredOutputs.every(p => record.outputs.some(o => o.path === p))) return false;
    for (const item of record.outputs) {
      const bytes = readEvidence(root, item); // reread ignored/untracked bytes, not HEAD
      if (!freshCheck(item.verification, item.sha256)) return false;
      const report = JSON.parse(bytes.toString('utf8'));
      if (!Array.isArray(report.criteriaSatisfied) || !Array.isArray(report.residualRisks) ||
          !Array.isArray(report.changedFiles) || !report.residualRisks.every(text) ||
          !report.changedFiles.every(text)) return false;
      for (const id of record.requiredCriteria) {
        const rows = report.criteriaSatisfied.filter(c => c.id === id);
        if (rows.length !== 1 || rows[0].status !== 'satisfied' || !text(rows[0].evidence)) return false;
      }
      evidence.push({ path: item.path, sha256: item.sha256, invocationId: item.verification.invocationId });
    }
    return true;
  });
  check('reviewsForBytes', () => {
    if (!Array.isArray(record.reviews) || !record.reviews.length || !Array.isArray(record.corrections)) return false;
    return record.outputs.every(item => {
      const reviewed = record.reviews.find(r => r.path === item.path && text(r.reviewer) &&
        r.independent === true && r.settled === true && digest(r.sha256));
      if (!reviewed) return false;
      if (reviewed.sha256 === item.sha256) return true;
      // Parent verification does NOT inherit independent review of corrected bytes.
      const correction = record.corrections.find(c => c.path === item.path && c.oldReviewedSha256 === reviewed.sha256 &&
        c.newSha256 === item.sha256 && text(c.disposition) && c.independentlyReviewed === false &&
        freshCheck(c.parentVerification, item.sha256));
      if (!correction) return false;
      corrections.push({ path: item.path, oldReviewedSha256: reviewed.sha256, newSha256: item.sha256,
        disposition: correction.disposition, independentlyReviewed: false,
        parentInvocationId: correction.parentVerification.invocationId });
      return true;
    });
  });
  check('reviewDisposition', () => {
    findings = canonicalFindings(record.findings);
    const open = findings.filter(f => f.unresolved);
    const medium = open.filter(f => f.severity === 'Medium');
    return !open.some(f => f.severity === 'Critical' || f.severity === 'Major') &&
      medium.length <= 2 && medium.every(f => f.residual !== null);
  });
  for (const field of ['requirements', 'tests', 'authority', 'release']) {
    check(`${field}Gate`, () => record.gates?.[field] === 'passed');
  }
  check('budgetNotExceeded', () => budget && integer(budget.reviewCycles) &&
    integer(budget.maxReviewCycles) && budget.maxReviewCycles <= 2 && budget.maxReviewCycles >= 1 &&
    budget.reviewCycles >= 1 && budget.reviewCycles <= budget.maxReviewCycles &&
    integer(budget.runsUsed) && integer(budget.maxRuns) && budget.runsUsed <= budget.maxRuns &&
    integer(budget.maxConcurrent) && budget.maxConcurrent >= 1 &&
    integer(budget.elapsedMs) && integer(budget.maxElapsedMs) && budget.elapsedMs <= budget.maxElapsedMs);
  // Capacity/headroom admit consuming actions, never block a clean closure.
  check('actionAdmission', () => {
    if (!request || !integer(request.runs) || !integer(request.reviewCycles) ||
        typeof request.expensive !== 'boolean' || request.reviewCycles > 1) return false;
    if (request.runs > budget.maxConcurrent ||
        ((request.runs > 0 || request.reviewCycles > 0 || request.expensive) && budget.elapsedMs >= budget.maxElapsedMs) ||
        budget.runsUsed + request.runs > budget.maxRuns ||
        budget.reviewCycles + request.reviewCycles > budget.maxReviewCycles ||
        budget.reviewCycles + request.reviewCycles > 2) return false;
    if (!request.expensive) return true;
    const m = record.memory;
    return m && Number.isFinite(m.freeMiB) && Number.isFinite(m.availableMiB) &&
      Number.isFinite(m.containerHeadroomMiB) && Number.isFinite(m.estimatedMiB) && m.estimatedMiB >= 0 &&
      m.freeMiB >= 750 && m.availableMiB >= 750 &&
      Math.min(m.freeMiB, m.availableMiB, m.containerHeadroomMiB) - m.estimatedMiB >= 500;
  });
  return { verdict: reasons.length ? 'blocked' : 'ready', reasons, predicates, findings, evidence, corrections };
}

// Arguments are supplied by the coordinator under a real launch grant. No
// filesystem or process API is exposed to the generated native workflow body.
function workflowExample({ lanes, synthesis, cwd }) {
  if (!text(cwd) || !Array.isArray(lanes) || lanes.length === 0 || !synthesis) throw new Error('missing launch inputs');
  const all = [...lanes, synthesis];
  if (new Set(all.map(l => l.key)).size !== all.length ||
      new Set(all.map(l => l.output)).size !== all.length) throw new Error('duplicate key/output');
  for (const item of all) {
    for (const key of ['key', 'label', 'agent', 'task', 'output', 'gateCommand']) {
      if (!text(item[key])) throw new Error(`missing ${key}`);
    }
  }
  const launch = item => ({ key: item.key, label: item.label, agent: item.agent,
    task: item.task, cwd, context: 'fresh', output: item.output, outputMode: 'file-only',
    gate: { command: item.gateCommand, output: 'json', schema } });
  return `
const lanes = ${JSON.stringify(lanes.map(launch))};
// runs.all preserves input order; observe all children even if one rejects.
const results = await runs.all(lanes);
function ready(result) {
  return result && result.ok === true && result.state !== 'running' &&
    result.structuredOutput && result.structuredOutput.verdict === 'ready' &&
    typeof result.outputReference === 'string' && result.outputReference.length > 0;
}
if (results.length !== lanes.length || results.some((r, i) => r.key !== lanes[i].key || !ready(r))) {
  return { verdict: 'blocked', phase: 'research', results };
}
const synthesis = ${JSON.stringify(launch(synthesis))};
// The sole synthesis author must READ saved bytes, not the file-only pointers.
synthesis.task += '\\nRead every completed input before synthesis: ' + JSON.stringify(results.map(r => r.outputReference));
const { key, ...params } = synthesis;
const result = await runs.run(key, params);
return { verdict: ready(result) ? 'ready' : 'blocked', phase: 'synthesis', inputs: results.map(r => r.outputReference), result };
`;
}

module.exports = { evaluateStage, workflowExample, sha256 };
