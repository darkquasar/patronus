import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, mkdtemp, mkdir, writeFile, rm, realpath, symlink } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { tmpdir } from 'node:os';
import { execFileSync } from 'node:child_process';
const dir = dirname(fileURLToPath(import.meta.url));
const script = await readFile(join(dir,'../workflow.js'),'utf8');
const schemaText = await readFile(join(dir,'../request.schema.json'),'utf8');
const fixture = JSON.parse(await readFile(join(dir,'request.json'),'utf8'));
const kind = fixture.reconcile !== undefined ? 'research' : fixture.baseRef ? 'implement' : 'peer-review';
const fresh = () => structuredClone(fixture);
const packageDir=process.env.PI_SUBAGENTS_PACKAGE;
const upstreamArgs=packageDir ? await import(pathToFileURL(join(packageDir,'src/workflows/workflow-resources.js'))) : null;
function normalized(args) {
  if (!upstreamArgs) return args;
  const result=upstreamArgs.normalizeWorkflowArgs(args);
  assert.equal(result.error,undefined,result.error);
  return result.args;
}
const evaluate = new (Object.getPrototypeOf(async function(){}).constructor)('args','runs',script);
async function execute(args,runs) { return evaluate(normalized(args),runs); }
function harness(change = () => {}) {
  const calls = [], waves = [];
  async function run(key,p) {
    calls.push({key,...p});
    assert.equal(p.async,undefined, 'await final results, not dispatch receipts');
    assert.equal(p.outputMode,p.agent.startsWith('patronus-') && p.outputSchema ? 'inline' : 'file-only');
    if (!p.agent.startsWith('patronus-')) for (const unsupported of ['context','model','tools','skill','acceptance','outputSchema','agentContract','toolBudget']) assert.equal(p[unsupported],undefined,unsupported);
    const runId='fixture-'+key, asyncDir='/fixture/runtime/'+runId;
    const output='Output saved to: '+p.output+' (16 B, 1 line). Read this file if needed.';
    const r = {key,agent:p.agent,ok:true,runId,asyncDir,continuation:{runIds:[runId]},output,outputReference:p.output,artifactPaths:[asyncDir],structuredOutput:{verdict:'clear',summary:'invented evidence'},results:[{index:0,agent:p.agent,exitCode:0,finalOutput:output}]};
    if (!p.agent.startsWith('patronus-')) {
      delete r.outputReference;
      r.externalAdapter={adapter:{id:p.agent,version:1,executionMode:'one-shot-stdin'},handoff:{mode:'fresh'},capabilities:{stop:true,resume:false,structuredOutput:false},outputArtifacts:{stdoutPath:asyncDir+'/external-0.stdout.log',stderrPath:asyncDir+'/external-0.stderr.log'}};
    }
    await change(r,p,calls);
    return r;
  }
  return {calls,waves,runs:{run,all:items => {waves.push(items.map(i=>i.key)); return Promise.all(items.map(({key,...p})=>run(key,p)));}}};
}
async function rejected(mutate) {
  const args=fresh();mutate(args);const h=harness();
  await assert.rejects(execute(args,h.runs));assert.equal(h.calls.length,0,'invalid request spawned children');
}
function optIn(operation='validate',level='parent-direct') {
  const a=fresh();
  a.preflight.snapshotUse={schemaVersion:1,snapshotId:'1'.repeat(32),path:'/fixture/capability-snapshot-v1.json',sha256:'2'.repeat(64),environmentIdentitySha256:'3'.repeat(64),observedAt:'2026-10-08T00:00:00Z'};
  a.preflight.liveChecks={checkedAt:'2026-10-08T00:01:00Z',authorityEvidenceSha256:'4'.repeat(64),resourceEvidenceSha256:'5'.repeat(64),outputClaimEvidenceSha256:'6'.repeat(64),settlementEvidenceSha256:'7'.repeat(64),selectedCapabilities:[{kind:'runtime',key:'native',evidenceSha256:'8'.repeat(64)}]};
  delete a.baseRef;delete a.integrationCwd;delete a.validationCommands;
  a.tasks=[];a.roles={};a.authorization.actions=['validate'];a.authorization.files=[];a.runners={native:a.runners.native};a.spawnLimit=1;
  const evidenceRef=(kind='target-delivery')=>({path:'/fixture/evidence/'+kind+'.json',sha256:'9'.repeat(64),subjectSha256:'a'.repeat(64),kind});
  const predecessor={key:'integrated',runId:'fixture-integration',agent:'patronus-writer-pi',report:evidenceRef('report'),handoff:evidenceRef('handoff'),delivery:evidenceRef()};
  a.execution={operation, integrationCwd:'/fixture/integration', predecessors:[predecessor], validation:{level,risk:{changedBehavior:'workflow admission',falsePassConsequence:'dependent work may start',determinism:'fixture exact',novelty:'new route',blastRadius:'workflow callers',costJustification:'existing seam'},question:'Does the exact subject pass?',acceptancePoint:'all checks pass',subjectContext:'invented exact subject',checks:[{key:'fixture',argv:['node','--test','fixture'],kind:'command'}]}};
  return a;
}
test('actual shipped body has portable sandbox syntax and explicit bindings', () => {
  assert.doesNotMatch(script,/\b(?:require|import|process|Buffer)\b|runs\.host|async\s+(?:function|\()/);
  assert.ok(JSON.parse(schemaText).required.includes('authorization'));
});
test('all stages execute with stable keys, durable evidence and parameter variation',async()=>{
  for (const suffix of ['one','two']) {
    const args=fresh();args.outputDir='/fixture/evidence/'+suffix;args.sourceRevision='revision-'+suffix;args.tasks[0].text='varied question '+suffix;
    const h=harness();const r=await execute(args,h.runs);
    assert.equal(r.verdict,'ready-for-parent');assert.equal(r.sourceRevision,args.sourceRevision);assert.equal(r.outputDir,args.outputDir);assert.equal(r.parentAcceptanceRequired,true);
    assert.ok(h.calls.every(c=>c.output.startsWith(args.outputDir+'/') && c.task.includes(args.sourceRevision)));
    assert.ok(h.calls.some(c=>c.task.includes(args.tasks[0].text)));
    assert.equal(new Set(h.calls.map(c=>c.key)).size,h.calls.length);
    assert.deepEqual(r.selected,kind==='research'?['native']:Object.keys(args.runners));
    assert.equal(r.results.length,h.calls.length);assert.ok(r.results.every(c=>c.runId && c.outputBinding?.outputReference));
    if(kind==='research') assert.deepEqual(h.calls.map(c=>c.key),['research-local','research-web','spec','spec-and-plan','review-plan','review-security']);
    if(kind==='implement') {
      assert.deepEqual(h.calls.map(c=>c.key),['writer-native','writer-claude','writer-codex','integration','validation']);
      assert.ok(h.calls.slice(0,3).every(c=>c.worktree===true && c.baseRef===args.baseRef));
      assert.ok(h.calls.slice(3).every(c=>c.cwd===args.integrationCwd && c.worktree===false));
      assert.match(h.calls.at(-1).task,/test -f src\/a.txt/);
    }
    if(kind==='peer-review') assert.deepEqual(h.waves,[['peer-native','peer-claude-code','peer-codex-exec']]);
  }
});
test('durable native verdict stages do not combine structured output with file-only persistence',async()=>{
  const h=harness();await execute(fresh(),h.runs);
  const verdictCalls=h.calls.filter(c=>c.agent.startsWith('patronus-') && c.outputSchema);
  assert.ok(verdictCalls.length>0);
  assert.ok(verdictCalls.every(c=>c.outputMode==='inline' && c.output.startsWith(fixture.outputDir+'/')));
});
test('bounded task inventory variation is not a hardcoded demonstration',async()=>{
  const a=fresh();
  while(a.tasks.length<4) {const i=a.tasks.length;const task={...a.tasks[0],key:'extra-'+i,text:'Independent varied task '+i};if(kind==='implement'){task.files=['src/extra-'+i+'.txt'];a.authorization.files.push(...task.files);}a.tasks.push(task);}
  const h=harness();const r=await execute(a,h.runs);assert.equal(r.verdict,'ready-for-parent');assert.ok(h.calls.some(c=>c.task.includes('Independent varied task')));
});
test('malformed inputs, actual-grant assertion, effective roles and caps fail before dispatch',async()=>{
  for(const mutate of [a=>{delete a.authorization},a=>{a.authorization.parentVerified=false},a=>{a.authorization.actions=[]},a=>{a.preflight.rolesVerified=false},a=>{a.preflight.pathsVerified=false},a=>{a.runners.native.version='0.88.0'},a=>{a.preflight.subagentsVersion='0.74.0'},a=>{a.roles[Object.keys(a.roles)[0]]='worker'},a=>{a.unrecognized=true},a=>{a.evidence=[]},a=>{a.evidence[0].text='x'.repeat(16001)},a=>{a.tasks[0].key='../escape'},a=>{a.tasks.push(a.tasks[0])},a=>{a.spawnLimit=1},a=>{a.spawnLimit=13},a=>{a.concurrency=kind==='implement'?11:3},a=>{a.concurrency=0},a=>{a.timeoutMs=0},a=>{a.timeoutMs=3600001}]) await rejected(mutate);
});
function tenWriterRequest(count = 10) {
  const a=fresh();
  a.tasks=Array.from({length:count},(_,i)=>({key:'lane-'+i,text:'Independent native writer '+i,engine:'native',files:['src/lane-'+i+'.txt']}));
  a.authorization.files=a.tasks.flatMap(t=>t.files);
  a.concurrency=10;
  a.spawnLimit=12;
  return a;
}
if(kind==='implement') {
  test('parent-direct opt-in stops pending at the delivery barrier with zero validator launches',async()=>{
    const a=optIn();const h=harness();const result=await execute(a,h.runs);
    assert.equal(result.verdict,'awaiting-parent-validation');
    assert.equal(result.phase,'parent-direct');
    assert.equal(h.calls.length,0);
    assert.deepEqual(result.lifecycle.started,[]);
    assert.ok(result.lifecycle.pendingParent.some(x=>x.kind==='validation'));
    assert.deepEqual(result.preflight.snapshotUse,a.preflight.snapshotUse);
    assert.deepEqual(result.preflight.liveChecks,a.preflight.liveChecks);
  });
  test('opt-in operations are phase-separated and child validation receives only the bounded brief',async()=>{
    const write=optIn();write.execution={operation:'write',baseRef:'refs/tags/v2.6.0',predecessors:[]};write.tasks=[{key:'one',text:'Implement one behavior',engine:'native',files:['src/a.txt']}];write.roles={writer:fixture.roles.writer};write.authorization.actions=['implement','managed-cleanup'];write.authorization.files=['src/a.txt'];
    const wh=harness();const wr=await execute(write,wh.runs);assert.equal(wr.verdict,'awaiting-parent-target-delivery');assert.deepEqual(wh.calls.map(c=>c.key),['writer-one']);assert.equal(wr.lifecycle.pendingParent[0].kind,'target-delivery');
    const integrate=optIn();integrate.execution.operation='integrate';integrate.execution.baseRef='refs/tags/v2.6.0';delete integrate.execution.validation;integrate.roles={integrator:fixture.roles.integrator};integrate.authorization.actions=['integrate'];
    const ih=harness();const ir=await execute(integrate,ih.runs);assert.equal(ir.verdict,'awaiting-parent-target-delivery');assert.deepEqual(ih.calls.map(c=>c.key),['integration']);
    for(const level of ['bounded-child','deep']) {
      const a=optIn('validate',level);a.roles={validator:fixture.roles.validator};if(level==='deep'){a.execution.validation.priorShortfall={path:'/fixture/shortfall.json',sha256:'b'.repeat(64),subjectSha256:'a'.repeat(64),kind:'shortfall'};a.execution.validation.approvalEvidence='owner approved exact deep question';}
      const h=harness();const r=await execute(a,h.runs);assert.equal(r.verdict,'ready-for-parent');assert.deepEqual(h.calls.map(c=>c.key),['validation']);assert.match(h.calls[0].task,/Execute only this bounded validation brief/);assert.doesNotMatch(h.calls[0].task,/authorization|runners/);assert.equal(r.lifecycle.achieved[0].kind,'validation');
    }
  });
  test('opt-in matrices, delivery references and shortfalls fail closed before successors',async()=>{
    for(const mutate of [a=>{a.baseRef='refs/tags/v2.6.0'},a=>{delete a.preflight.snapshotUse},a=>{a.execution.predecessors[0].delivery.kind='report'},a=>{a.execution.validation.level='deep'},a=>{a.execution.validation.approvalEvidence='approval only'},a=>{a.roles={validator:fixture.roles.validator}},a=>{a.execution.validation.checks.at(-1).surprise=true}]) {
      const a=optIn();mutate(a);const h=harness();await assert.rejects(execute(a,h.runs));assert.equal(h.calls.length,0);
    }
    const a=optIn('validate','bounded-child');a.roles={validator:fixture.roles.validator};const h=harness(r=>{if(r.key==='validation')r.structuredOutput={outcome:'shortfall',observations:['invented'],evidenceAttempted:['fixture'],unresolved:['identity'],newRequirements:['compatibility evidence'],confidence:'low',consequences:'false pass risk'};});const result=await execute(a,h.runs);assert.equal(result.verdict,'awaiting-parent-shortfall');assert.equal(result.lifecycle.settled[0].outcome,'shortfall');assert.equal(result.lifecycle.achieved.length,0);assert.equal(h.calls.length,1);
  });
  function mixedNativeRequest() {
    const a=fresh();
    a.tasks=a.tasks.slice(0,2).map((t,i)=>({...t,engine:'native',model:['anthropic/claude-opus-4-8','openai-codex/gpt-6.1-sol:low'][i]}));
    return a;
  }
  test('per-writer native models forward exactly without changing role or stage defaults',async()=>{
    const a=mixedNativeRequest();const h=harness();const result=await execute(a,h.runs);
    assert.equal(result.verdict,'ready-for-parent');assert.deepEqual(result.selected,['native']);
    assert.deepEqual(h.calls.slice(0,2).map(c=>c.model),a.tasks.map(t=>t.model));
    assert.ok(h.calls.every(c=>c.agent.startsWith('patronus-')),'native model selection must not launch CLI engines');
    for(const c of h.calls.slice(0,2)) {
      assert.equal(c.agent,a.roles.writer);assert.equal(c.worktree,true);assert.equal(c.baseRef,a.baseRef);assert.equal(c.context,'fresh');
      assert.equal(result.results.find(r=>r.key===c.key).outputBinding.requestedModel,c.model);
    }
    assert.ok(h.calls.slice(2).every(c=>!Object.hasOwn(c,'model')),'writer models leaked to integration/validation');
    const legacy=harness();const unchanged=await execute(fresh(),legacy.runs);
    assert.equal(unchanged.verdict,'ready-for-parent');
    assert.ok(legacy.calls.every(c=>!Object.hasOwn(c,'model')),'omitted models must preserve role defaults and external contracts');
    assert.ok(unchanged.results.every(r=>r.outputBinding.requestedModel===null));
    const failed=harness(r=>{if(r.key==='writer-native'){r.ok=false;r.error='invented selected-model authentication failure';}});
    const blocked=await execute(a,failed.runs);
    assert.equal(blocked.verdict,'blocked');assert.equal(blocked.phase,'writers');
    assert.equal(failed.calls.length,2,'failed selected model was retried/replaced or advanced dependencies');
    assert.deepEqual(failed.calls.map(c=>c.model),a.tasks.map(t=>t.model));
  });
  test('native model syntax and external model rejection agree with schema before dispatch',async()=>{
    const item=JSON.parse(schemaText).properties.tasks.items;
    assert.ok(!item.required.includes('model'));assert.equal(item.properties.model.maxLength,200);
    assert.deepEqual(item.allOf,[{if:{required:['model']},then:{properties:{engine:{const:'native'}}}}]);
    const pattern=new RegExp(item.properties.model.pattern);
    for(const model of ['openai-codex/gpt-6.1-sol','anthropic/claude-opus-4-8:high','huggingface/owner/model']) {
      assert.ok(pattern.test(model));
      const a=fresh();a.tasks[0].model=model;const h=harness();assert.equal((await execute(a,h.runs)).verdict,'ready-for-parent');assert.equal(h.calls[0].model,model);
    }
    for(const model of ['',' ','inherit','gpt-6.1-sol','/id','provider/','provider//id','provider/id:unsupported','provider/id:high:low','provider/id\n',' provider/id','provider/id ','provider/id with spaces','provider/'+'x'.repeat(192)]) {
      if(model.length<=200)assert.equal(pattern.test(model),false,JSON.stringify(model));
      await rejected(a=>{a.tasks[0].model=model;});
    }
    for(const model of [null,false,3,{},[]])await rejected(a=>{a.tasks[0].model=model;});
    for(const engine of ['claude-code-writer','codex-exec-writer'])await rejected(a=>{a.tasks[0].engine=engine;a.tasks[0].model='anthropic/claude-opus-4-8';});
    await rejected(a=>{a.tasks.at(-1).model='anthropic/claude-opus-4-8';});
  });
  test('real pinned sandbox keeps mixed native models per writer and model failures block dependencies',{skip:!packageDir},async()=>{
    const {validateWorkflowScript,runWorkflowScript}=await import(pathToFileURL(join(packageDir,'src/workflows/scripted-workflow.js')));
    assert.deepEqual(validateWorkflowScript(script).errors,[]);
    const a=mixedNativeRequest();
    for(const failure of [false,true]) {
      const h=harness(r=>{if(failure&&r.key==='writer-native'){r.ok=false;r.error='invented model unavailable';}});
      const result=await runWorkflowScript({script,args:normalized(a),globalConcurrencyLimit:2,timeoutMs:10000,launch:h.runs.run,status:async()=>{throw new Error('unexpected polling')}});
      assert.equal(result.value.verdict,failure?'blocked':'ready-for-parent');
      assert.deepEqual(h.calls.slice(0,2).map(c=>c.model),a.tasks.map(t=>t.model));
      assert.equal(h.calls.length,failure?2:4);assert.equal(result.children.length,h.calls.length);
      assert.ok(h.calls.slice(2).every(c=>!Object.hasOwn(c,'model')));
    }
    const bad=mixedNativeRequest();bad.tasks[1].engine='claude-code-writer';const denied=harness();
    await assert.rejects(runWorkflowScript({script,args:normalized(bad),globalConcurrencyLimit:2,timeoutMs:10000,launch:denied.runs.run,status:async()=>{throw new Error('unexpected polling')}}),/model override requires a native writer/);
    assert.equal(denied.calls.length,0);
  });
  test('ten writer ceiling matches schema and reserves integration and validation',async()=>{
    const schema=JSON.parse(schemaText);
    assert.equal(schema.properties.concurrency.maximum,10);
    assert.equal(schema.properties.tasks.maxItems,10);
    const a=tenWriterRequest();const h=harness();const result=await execute(a,h.runs);
    assert.equal(result.verdict,'ready-for-parent');
    assert.equal(result.limits.concurrency,10);
    assert.equal(h.waves[0].length,10);
    assert.equal(h.calls.length,12);
    assert.deepEqual(h.calls.slice(-2).map(c=>c.key),['integration','validation']);
    for(const mutate of [a=>{a.concurrency=11},a=>{a.spawnLimit=11},a=>{a.tasks=tenWriterRequest(11).tasks;a.authorization.files=a.tasks.flatMap(t=>t.files)}]) {
      const bad=tenWriterRequest();mutate(bad);const denied=harness();
      await assert.rejects(execute(bad,denied.runs));assert.equal(denied.calls.length,0);
    }
    const failed=harness(r=>{if(r.key==='writer-lane-8')r.ok=false});
    assert.equal((await execute(a,failed.runs)).verdict,'blocked');
    assert.equal(failed.calls.length,10,'failed writer must not advance dependent stages');
  });
  test('real pinned sandbox admits ten concurrent writers without changing the stage barriers',{skip:!packageDir},async()=>{
    const {validateWorkflowScript,runWorkflowScript}=await import(pathToFileURL(join(packageDir,'src/workflows/scripted-workflow.js')));
    assert.deepEqual(validateWorkflowScript(script).errors,[]);
    const a=tenWriterRequest();const h=harness();let active=0,peak=0;
    const result=await runWorkflowScript({script,args:normalized(a),globalConcurrencyLimit:a.concurrency,timeoutMs:10000,
      launch:async(key,p)=>{
        if(key==='integration'||key==='validation')assert.equal(active,0,'dependent stage overlapped writers');
        active++;peak=Math.max(peak,active);
        try {await new Promise(resolve=>setTimeout(resolve,5));return await h.runs.run(key,p);} finally {active--;}
      },status:async()=>{throw new Error('unexpected polling')}});
    assert.equal(result.value.verdict,'ready-for-parent');assert.equal(peak,10);
    assert.equal(result.children.length,12);assert.equal(active,0);
  });
}
test('source-root, integration-root and lexical unsafe output rejection',async()=>{
  for(const output of ['/fixture/source/report','/fixture/integration/report','/fixture/worktrees/report','/fixture','relative','/fixture/evidence/../source/out','/fixture//out','/fixture/./out','/fixture/out/','/fixture/out\\bad']) await rejected(a=>{a.outputDir=output});
  await rejected(a=>{a.sourceRoots=[]});
});
test('every runner candidate status is handled without silent replacement',async()=>{
  for(const engine of Object.keys(fixture.runners)) {
    await rejected(a=>{delete a.runners[engine]});
    await rejected(a=>{a.runners[engine].status='unknown'});
    await rejected(a=>{a.runners[engine].status='available';a.runners[engine].authEvidence=''});
    await rejected(a=>{a.runners[engine].contractVerified=false});
    await rejected(a=>{a.runners[engine].executable='pi'});
    if(kind!=='peer-review' || engine==='native') await rejected(a=>{a.runners[engine].status='unavailable'});
    const a=fresh();Object.assign(a.runners[engine],{status:'unverified',authEvidence:'UNVERIFIED',allowUnverified:true});
    const h=harness();const r=await execute(a,h.runs);assert.ok(r.selected.includes(engine));assert.equal(r.runnerEvidence[engine].status,'unverified');
    a.runners[engine].allowUnverified=false;
    if(kind!=='peer-review' || engine==='native') { const h2=harness();await assert.rejects(execute(a,h2.runs));assert.equal(h2.calls.length,0); }
    else {const r2=await execute(a,harness().runs);assert.ok(r2.omitted.some(o=>o.engine===engine));}
  }
  await rejected(a=>{a.runners.surprise={...a.runners.native}});
});
test('failure/stop/dispatch receipt/missing output blocks all dependent stages, siblings settle',async()=>{
  const success=harness();await execute(fresh(),success.runs);
  for(const key of success.calls.map(c=>c.key)) for(const fault of ['failure','stopped','running','detached','timeout','missing-output','wrong-output','missing-run']) {
    const h=harness(r=>{if(r.key!==key)return;if(fault==='failure')r.ok=false;if(fault==='stopped')r.stopped=true;if(fault==='running')r.state='running';if(fault==='detached')r.detached=true;if(fault==='timeout')r.terminalOutcome={state:'partial',reason:'timeout'};if(fault==='missing-output'){delete r.outputReference;delete r.results;}if(fault==='wrong-output')r.outputReference='/fixture/source/report';if(fault==='missing-run')delete r.runId;});
    const r=await execute(fresh(),h.runs);assert.equal(r.verdict,'blocked',key+' '+fault);assert.ok(h.calls.length<=success.calls.length);assert.equal(h.calls.filter(c=>c.key===key).length,1);
    if(success.calls.findIndex(c=>c.key===key)<(kind==='research'?2:3)) assert.ok(!h.calls.some(c=>['spec','integration','disposition'].includes(c.key)));
  }
});
test('duplicate accepted run IDs block before dependent stages',async()=>{
  let firstRunId;
  const h=harness(r=>{if(firstRunId===undefined)firstRunId=r.runId;else r.runId=firstRunId;});
  const r=await execute(fresh(),h.runs);assert.equal(r.verdict,'blocked');
  const dependent=kind==='research'?'spec':kind==='implement'?'integration':'disposition';
  assert.ok(!h.calls.some(c=>c.key===dependent));
});
test('rejected runner promise cannot advance dependencies',async()=>{
  const h=harness();h.runs.all=items=>Promise.all(items.map(({key,...p},i)=>i===0?Promise.reject(new Error('fixture runner failure')):h.runs.run(key,p)));
  await assert.rejects(execute(fresh(),h.runs),/fixture runner failure/);
  assert.ok(!h.calls.some(c=>['spec','integration','disposition'].includes(c.key)));
});
if(kind==='research') {
  test('one reconciliation only, never recursive review, unauthorized correction blocks',async()=>{
    const h=harness(r=>{if(r.key.startsWith('review-'))r.structuredOutput.verdict='reconcile'});
    const r=await execute(fresh(),h.runs);assert.equal(r.reconciliations,1);assert.equal(r.independentlyReviewedFinalBytes,false);assert.equal(h.calls.at(-1).key,'reconciliation');assert.equal(h.calls.length,7);
    const a=fresh();a.reconcile=false;assert.equal((await execute(a,harness(r=>{if(r.key.startsWith('review-'))r.structuredOutput.verdict='reconcile'}).runs)).verdict,'blocked');
    for(const verdict of ['blocked','invalid',undefined]) {const h=harness(r=>{if(r.key==='review-plan')r.structuredOutput.verdict=verdict});assert.equal((await execute(fresh(),h.runs)).verdict,'blocked');assert.ok(!h.calls.some(c=>c.key==='reconciliation'));}
    await rejected(a=>{a.tasks[1].lane='local'});await rejected(a=>{a.reconcile=2});
  });
}
if(kind==='implement') {
  test('all external writer choices remain selectable; overlap and denied coding fail predispatch',async()=>{
    for(const engine of Object.keys(fixture.runners)) {const a=fresh();a.tasks=[{key:'one',text:'Write A',files:['src/a.txt'],engine}];const h=harness();await execute(a,h.runs);assert.equal(h.calls[0].agent,engine==='native'?a.roles.writer:engine);assert.equal(h.calls[0].worktree,true);}
    for(const mutate of [a=>{a.tasks[1].files=['src/a.txt']},a=>{a.tasks[0].files=['src']},a=>{a.tasks[0].files=['../out']},a=>{a.tasks[0].engine='claude-code'},a=>{a.authorization.files=[]},a=>{a.authorization.actions=['review']},a=>{a.integrationCwd=a.cwd},a=>{a.baseRef='refs/heads/a/.hidden'},a=>{a.baseRef='refs/heads/a..b'},a=>{a.baseRef='refs/heads/a//b'},a=>{a.baseRef='refs/heads/a.lock/b'},a=>{a.baseRef='HEAD~1'},a=>{a.baseRef='a'.repeat(40)},a=>{a.validationCommands=[]}]) await rejected(mutate);
    const h=harness(r=>{if(r.key==='writer-claude')r.artifactPaths=[]});assert.equal((await execute(fresh(),h.runs)).verdict,'blocked');assert.ok(!h.calls.some(c=>c.key==='integration'));
    for(const key of ['integration','validation']) {const h=harness(r=>{if(r.key===key)r.structuredOutput.verdict='blocked'});assert.equal((await execute(fresh(),h.runs)).verdict,'blocked');}
  });
}
if(kind==='peer-review') {
  test('omissions and expressly reduced review, no replacement of failed selected peer',async()=>{
    const a=fresh();a.runners['codex-exec'].status='unavailable';a.runners['codex-exec'].reason='missing executable';
    const r=await execute(a,harness().runs);assert.deepEqual(r.requested,['native','claude-code','codex-exec']);assert.deepEqual(r.selected,['native','claude-code']);assert.equal(r.omitted[0].reason,'missing executable');assert.equal(r.reduced,false);
    a.runners['claude-code'].status='unavailable';await assert.rejects(execute(a,harness().runs));a.allowReduced=true;const reduced=await execute(a,harness().runs);assert.equal(reduced.reduced,true);assert.equal(reduced.omitted.length,2);
    const h=harness(r=>{if(r.key==='peer-claude-code')r.ok=false});const blocked=await execute(fresh(),h.runs);assert.equal(blocked.verdict,'blocked');assert.equal(h.calls.length,3);assert.equal(blocked.omitted.length,0);
  });
}
test('exact installed 0.72.1 validator and real sandbox execute shipped body with model-free final results',{skip:!packageDir},async()=>{
  assert.equal(JSON.parse(await readFile(join(packageDir,'package.json'),'utf8')).version,'0.72.1');
  const {validateWorkflowScript,runWorkflowScript}=await import(pathToFileURL(join(packageDir,'src/workflows/scripted-workflow.js')));
  assert.deepEqual(validateWorkflowScript(script).errors,[]);
  const h=harness();let active=0,peak=0;
  const result=await runWorkflowScript({script,args:normalized(fresh()),globalConcurrencyLimit:2,timeoutMs:10000,launch:async(key,p)=>{active++;peak=Math.max(peak,active);await new Promise(r=>setTimeout(r,5));const r=await h.runs.run(key,p);active--;return r},status:async()=>{throw new Error('unexpected polling')}});
  assert.equal(result.value.verdict,'ready-for-parent');assert.equal(peak,2);assert.ok(result.children.length>0);
});
test('real sandbox timeout/abort stops admission and preserves partial child evidence',{skip:!packageDir},async()=>{
  const {runWorkflowScript}=await import(pathToFileURL(join(packageDir,'src/workflows/scripted-workflow.js')));
  const calls=[];
  await assert.rejects(runWorkflowScript({script,args:normalized(fresh()),globalConcurrencyLimit:1,timeoutMs:100,
    launch:(key,p,signal)=>new Promise((resolve,reject)=>{calls.push(key);const abort=()=>reject(new Error('fixture acknowledged abort'));if(signal.aborted)abort();else signal.addEventListener('abort',abort,{once:true});}),
    status:async()=>{throw new Error('unexpected polling')}}),error=>{assert.match(error.message,/timeout|timed out|abort/i);assert.ok(error.partial);return true;});
  assert.equal(calls.length,1);assert.ok(!calls.some(key=>['spec','integration','disposition'].includes(key)));
});
test('real pinned Git admission after persisted output and cleanup; symlink parent observations',{skip:!packageDir || kind!=='implement'},async()=>{
  const {preflightWorktreeSource}=await import(pathToFileURL(join(packageDir,'src/runs/shared/worktree.js')));
  const root=await mkdtemp(join(process.env.C23_SCRATCH || tmpdir(),'workflow-git-'));
  const source=join(root,'source'), evidence=join(root,'evidence');await mkdir(source);await mkdir(evidence);
  const git=(...args)=>execFileSync('git',['-C',source,...args],{encoding:'utf8',env:{...process.env,GIT_AUTHOR_DATE:'2026-10-03T18:00:00+10:00',GIT_COMMITTER_DATE:'2026-10-03T18:00:00+10:00'}});
  try {
    git('init','-q');git('config','user.name','Fixture');git('config','user.email','fixture@example.invalid');await writeFile(join(source,'base'),'base');git('add','base');git('commit','-qm','fixture');
    await preflightWorktreeSource(source);const wt=join(root,'writer');git('worktree','add','--detach',wt,'HEAD');
    await writeFile(join(evidence,'writer.md'),'persisted first report');await preflightWorktreeSource(source);git('worktree','remove',wt);const second=join(root,'second-writer');git('worktree','add','--detach',second,'HEAD');await preflightWorktreeSource(source);git('worktree','remove',second);
    assert.equal(await readFile(join(evidence,'writer.md'),'utf8'),'persisted first report');await preflightWorktreeSource(source);
    await writeFile(join(source,'bad-output.md'),'pollutes source');await assert.rejects(preflightWorktreeSource(source),/clean git working tree/);
    await symlink(source,join(root,'alias'));assert.equal(await realpath(join(root,'alias')),await realpath(source));
    const a=fresh();a.cwd=await realpath(source);a.sourceRoots=[a.cwd,a.integrationCwd];a.outputDir=join(await realpath(join(root,'alias')),'out');const h=harness();await assert.rejects(execute(a,h.runs),/output must be outside/);assert.equal(h.calls.length,0);
  } finally {await rm(root,{recursive:true,force:true});}
});
test('real upstream argument envelope, sentinel, UTF-8, depth, properties and arrays',{skip:!packageDir},async()=>{
  const {normalizeWorkflowArgs}=upstreamArgs;
  assert.ok(normalizeWorkflowArgs(fresh()).args);
  const unverified=fresh();Object.assign(unverified.runners.native,{status:'unverified',authEvidence:'UNVERIFIED',allowUnverified:true});
  assert.ok(normalizeWorkflowArgs(unverified).args);assert.equal((await execute(unverified,harness().runs)).verdict,'ready-for-parent');
  unverified.runners.native.allowUnverified=false;assert.ok(normalizeWorkflowArgs(unverified).args);await assert.rejects(execute(unverified,harness().runs));
  const empty=fresh();empty.runners.native.authEvidence='';assert.match(normalizeWorkflowArgs(empty).error,/empty/);
  const unicode=fresh();unicode.evidence[0].text='界'.repeat(1000);assert.ok(normalizeWorkflowArgs(unicode).args);assert.equal((await execute(unicode,harness().runs)).verdict,'ready-for-parent');
  const large=fresh();large.evidence[0].text='界'.repeat(6000);assert.match(normalizeWorkflowArgs(large).error,/bytes/);
  const boundary={x:'x'.repeat(16376)};assert.equal(Buffer.byteLength(JSON.stringify(boundary)),16384);assert.ok(normalizeWorkflowArgs(boundary).args);boundary.x+='x';assert.match(normalizeWorkflowArgs(boundary).error,/bytes/);
  const total={a:'界'.repeat(3000),b:'界'.repeat(3000)};assert.match(normalizeWorkflowArgs(total).error,/bytes/);
  let nested='x';for(let i=0;i<9;i++)nested={a:nested};assert.match(normalizeWorkflowArgs(nested).error,/nested/);
  assert.match(normalizeWorkflowArgs(Object.fromEntries(Array.from({length:17},(_,i)=>['k'+i,'x']))).error,/fields/);
  assert.match(normalizeWorkflowArgs({items:Array(65).fill('x')}).error,/items/);
  const evidenceTooLarge=fresh();evidenceTooLarge.evidence[0].text='界'.repeat(2800);assert.ok(normalizeWorkflowArgs(evidenceTooLarge).args);await assert.rejects(execute(evidenceTooLarge,harness().runs),/inline evidence/);
  if(process.env.C23_REQUESTS) for(const name of ['research','implement','peer-review']) {const prepared=JSON.parse(await readFile(join(process.env.C23_REQUESTS,name+'.json'),'utf8'));assert.ok(normalizeWorkflowArgs(prepared).args,JSON.stringify(normalizeWorkflowArgs(prepared)));}
});
if(kind!=='research') {
  test('external compatibility requires structured runtime evidence, exact identity, and no save/artifact failures',async()=>{
    const externalKey=kind==='implement'?'writer-claude':'peer-claude-code';
    const success=await execute(fresh(),harness().runs);const row=success.results.find(r=>r.key===externalKey);
    assert.equal(row.outputReference,null);assert.equal(row.outputBinding.filesystemVerified,false);assert.match(row.outputBinding.provenance,/structured-terminal/);assert.equal(row.outputBinding.metadataPersistence,'unverified-runtime-field-not-forwarded');assert.ok(row.underlyingResults.length===1);
    const faults=[r=>{delete r.externalAdapter},r=>{delete r.results},r=>{r.results.push(r.results[0])},r=>{r.results[0].exitCode=1},r=>{r.results[0].outputSaveError='actual write failure'},r=>{r.results[0].artifactOutputSaveFailed=true},r=>{r.results[0].metadataSaveError='metadata failure'},r=>{r.results[0].error='adapter failure'},r=>{r.results[0].timedOut=true},r=>{r.results[0].execution={status:'partial',success:false}},r=>{r.results[0].agent='codex-exec'},r=>{r.agent='wrong'},r=>{r.key='wrong'},r=>{r.runId='wrong'},r=>{r.results[0].runId='wrong'},r=>{r.results[0].workflowKey='wrong'},r=>{r.outputPathMapping={requestedPath:'/wrong',savedPath:'/wrong'}},r=>{r.results[0].savedOutputPath='/wrong'},r=>{r.externalAdapter.adapter.id='wrong'},r=>{r.externalAdapter.outputArtifacts.stdoutPath='/foreign/run/external-0.stdout.log'},r=>{r.results[0].finalOutput='model says saved'},r=>{r.output+='\nmodel suffix'},r=>{r.stopped=true},r=>{r.detached=true}];
    for(const fault of faults) {const h=harness(r=>{if(r.key===externalKey)fault(r)});const result=await execute(fresh(),h.runs);assert.equal(result.verdict,'blocked');assert.ok(!h.calls.some(c=>['integration','disposition'].includes(c.key)));}
  });
  test('real single-output save failure cannot be promoted by spoofed saved-summary text',{skip:!packageDir},async()=>{
    const {resolveSingleOutput,finalizeSingleOutput}=await import(pathToFileURL(join(packageDir,'src/runs/shared/single-output.js')));
    const root=await mkdtemp(join(process.env.C23_SCRATCH||tmpdir(),'save-proof-'));
    try {
      const path=join(root,'saved.md');const resolved=resolveSingleOutput(path,'actual fixture bytes');assert.equal(await readFile(path,'utf8'),'actual fixture bytes');assert.equal(resolved.savedPath,path);
      const finalized=finalizeSingleOutput({fullOutput:resolved.fullOutput,outputPath:path,outputMode:'file-only',exitCode:0,savedPath:resolved.savedPath});assert.match(finalized.displayOutput,/Output saved to:/);
      await writeFile(join(root,'not-a-directory'),'x');
      const failure=resolveSingleOutput(join(root,'not-a-directory','report.md'),'Output saved to: spoof');assert.ok(failure.saveError);assert.equal(failure.savedPath,undefined);
      const h=harness(r=>{if(r.externalAdapter)r.results[0].outputSaveError=failure.saveError});
      assert.equal((await execute(fresh(),h.runs)).verdict,'blocked');assert.ok(!h.calls.some(c=>['integration','disposition'].includes(c.key)));
    } finally {await rm(root,{recursive:true,force:true});}
  });
}
if(kind==='implement') {
  test('handoff candidates must be run-associated, and all actual captures are inspected before native integration',async()=>{
    for(const fault of [r=>{r.artifactPaths=['/arbitrary/report.md'];delete r.asyncDir},r=>{r.artifactPaths.push('/foreign/handoff.json')},r=>{r.artifactPaths=['/arbitrary/handoffs/foreign.json'];delete r.asyncDir}]) {
      const h=harness(r=>{if(r.key==='writer-native')fault(r)});assert.equal((await execute(fresh(),h.runs)).verdict,'blocked');assert.ok(!h.calls.some(c=>c.key==='integration'));
    }
    const foreground=harness(r=>{if(r.key==='writer-native'){delete r.asyncDir;const path='/fixture/handoffs/'+r.runId+'.json';r.artifactPaths=[path];r.outputArtifactPath=path;}});
    assert.equal((await execute(fresh(),foreground.runs)).verdict,'ready-for-parent');assert.match(foreground.calls.find(c=>c.key==='integration').task,/runtime-parallelHandoff/);
    // Capture bytes are not visible inside the sandbox. This models the native
    // integrator's required all-input read barrier, not actual model compliance.
    for(const capture of ['missing','partial','complete']) {
      const h=harness((r,p)=>{if(r.key==='integration') {
        assert.match(p.task,/Before ANY mutation/);assert.match(p.task,/Validate every handoff before applying ANY patch/);assert.match(p.task,/fixture-writer-native\/handoff\.json/);assert.match(p.task,/captureVerified":false/);assert.match(p.task,/patch.error/);
        if(capture!=='complete')r.structuredOutput.verdict='blocked';
      }});
      const result=await execute(fresh(),h.runs);assert.equal(result.verdict,capture==='complete'?'ready-for-parent':'blocked');assert.equal(h.calls.some(c=>c.key==='validation'),capture==='complete');
    }
  });
}
if(kind==='implement') test('native consumer barrier reads real invented manifests/patches using pinned identity resolver',{skip:!packageDir},async()=>{
  const {resolveParallelHandoffChild}=await import(pathToFileURL(join(packageDir,'src/runs/shared/parallel-handoff.js')));
  for(const capture of ['complete','missing','partial','foreign']) {
    const root=await mkdtemp(join(process.env.C23_SCRATCH||tmpdir(),'capture-proof-'));let mutations=0;
    try {
      const a=fresh();a.sourceRevision='a'.repeat(40);
      const h=harness(async(r,p)=>{
        if(r.key.startsWith('writer-')) {
          r.asyncDir=join(root,r.runId);await mkdir(r.asyncDir);r.artifactPaths=[r.asyncDir];
          if(r.externalAdapter)r.externalAdapter.outputArtifacts={stdoutPath:r.asyncDir+'/external-0.stdout.log',stderrPath:r.asyncDir+'/external-0.stderr.log'};
          const lane=a.tasks.find(t=>'writer-'+t.key===r.key), patchPath=join(r.asyncDir,'captured.patch');
          await writeFile(patchPath,'diff --git a/'+lane.files[0]+' b/'+lane.files[0]+'\n');
          const child={index:0,taskIndex:0,agent:r.agent,workflowKey:r.key,runId:r.runId,status:'completed',patch:{path:patchPath}};
          if(capture==='partial'&&r.key==='writer-native')child.patch.error='diff artifact unavailable; no patch was captured';
          const manifest={version:1,runId:capture==='foreign'&&r.key==='writer-native'?'foreign':r.runId,mode:'single',source:'async',groups:[{stepIndex:0,baseCommit:a.sourceRevision,repoRoot:a.cwd,children:[child],cleanup:{state:'complete',tasks:[{index:0}]}}]};
          if(!(capture==='missing'&&r.key==='writer-native'))await writeFile(join(r.asyncDir,'handoff.json'),JSON.stringify(manifest));
        }
        if(r.key==='integration') {
          // This native-consumer double performs the mandated read barrier; it is
          // not sandbox FS access or proof a live native agent follows the prompt.
          const candidates=JSON.parse(p.task.split('Exact handoff candidates: ')[1].split('\nApproved claims: ')[0]);
          try {
            for(const c of candidates) {
              const found=resolveParallelHandoffChild({manifestPath:c.candidate.path,runId:c.runId,workflowKey:c.key,childRunId:c.runId});
              assert.ok(found,'missing capture');assert.equal(found.child.agent,c.agent);assert.equal(found.child.status,'completed');assert.equal(found.group.baseCommit,a.sourceRevision);assert.equal(found.group.repoRoot,a.cwd);assert.equal(found.child.patch.error,undefined);
              const patch=await readFile(found.child.patch.path,'utf8');assert.equal(patch,'diff --git a/'+c.files[0]+' b/'+c.files[0]+'\n');
            }
            mutations++; // only after ALL actual bytes pass the barrier
          } catch {r.structuredOutput.verdict='blocked';}
        }
      });
      const result=await execute(a,h.runs);assert.equal(mutations,capture==='complete'?1:0);assert.equal(result.verdict,capture==='complete'?'ready-for-parent':'blocked');assert.equal(h.calls.some(c=>c.key==='validation'),capture==='complete');
    } finally {await rm(root,{recursive:true,force:true});}
  }
});

test('real sandbox receives normalized unverified granted/denied and prepared denied packets',{skip:!packageDir},async()=>{
  const {runWorkflowScript}=await import(pathToFileURL(join(packageDir,'src/workflows/scripted-workflow.js')));
  const a=fresh();Object.assign(a.runners.native,{status:'unverified',authEvidence:'UNVERIFIED',allowUnverified:true});
  const h=harness();const result=await runWorkflowScript({script,args:normalized(a),globalConcurrencyLimit:a.concurrency,timeoutMs:10000,launch:h.runs.run,status:async()=>{throw new Error('no polling')}});
  assert.equal(result.value.verdict,'ready-for-parent');
  a.runners.native.allowUnverified=false;const denied=harness();
  await assert.rejects(runWorkflowScript({script,args:normalized(a),globalConcurrencyLimit:a.concurrency,timeoutMs:10000,launch:denied.runs.run,status:async()=>{throw new Error('no polling')}}),/native runner/);assert.equal(denied.calls.length,0);
  if(process.env.C23_REQUESTS) {
    const prepared=JSON.parse(await readFile(join(process.env.C23_REQUESTS,kind+'.json'),'utf8'));const p=harness();
    await assert.rejects(runWorkflowScript({script,args:normalized(prepared),globalConcurrencyLimit:prepared.concurrency,timeoutMs:10000,launch:p.runs.run,status:async()=>{throw new Error('no polling')}}),/stage\/action grant/);assert.equal(p.calls.length,0);
  }
});
