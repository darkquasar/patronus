#!/usr/bin/env bash
# Offline contract tests: the only AWS executable available to the publisher is fake.
set -euo pipefail
root=$(cd "$(dirname "$0")/../.." && pwd)
python3 - "$root" <<'PY'
import hashlib, json, os, pathlib, subprocess, sys, tempfile
root = pathlib.Path(sys.argv[1])
with tempfile.TemporaryDirectory(prefix='publish packages ') as tmp:
    base = pathlib.Path(tmp)
    binpath = base / 'bin'
    binpath.mkdir()
    fake = binpath / 'aws'
    fake.write_text('''#!/usr/bin/env python3
import fcntl, hashlib, json, os, pathlib, sys, time
args = sys.argv[1:]
op = args[1]
def arg(name): return args[args.index(name)+1]
key = arg('--key')
store = pathlib.Path(os.environ['FAKE_STORE'])
f = store / key
with (store / 'calls').open('a') as log: log.write(op+' '+key+'\\n')
mode = os.environ.get('FAKE_MODE', '')
def error(code):
    print('An error occurred ('+code+') when calling the '+{'head-object':'HeadObject','put-object':'PutObject','get-object':'GetObject'}[op]+' operation: fake', file=sys.stderr)
    sys.exit(255)
if op == 'head-object':
    if mode == 'concurrent' and key.endswith('.tar.gz') and not (store/('seen-'+os.environ['PUBLISHER'])).exists():
        (store/('seen-'+os.environ['PUBLISHER'])).touch()
        deadline = time.monotonic()+10
        while len(list(store.glob('seen-*'))) < 2:
            assert time.monotonic() < deadline, 'second publisher did not reach HEAD'
            time.sleep(.01)
        error('404')
    if mode in ('403','timeout','misleading404'):
        error({'403':'403','timeout':'RequestTimeout','misleading404':'AccessDenied'}[mode])
    if not f.exists(): error('NoSuchKey' if mode == 'nosuchkey' else '404')
    print(json.dumps({'Metadata': json.loads(pathlib.Path(str(f)+'.meta').read_text())}))
elif op == 'get-object':
    pathlib.Path(args[-1]).write_bytes(f.read_bytes())
    print('{}')
elif op == 'put-object':
    lock = (store/'put-lock').open('w')
    fcntl.flock(lock, fcntl.LOCK_EX)
    if key.startswith('packages/'):
        assert arg('--if-none-match') == '*', 'unconditional package overwrite'
    if mode == 'interrupt' and key.endswith('.sha256'): error('RequestTimeout')
    f.parent.mkdir(parents=True, exist_ok=True)
    data = pathlib.Path(arg('--body')).read_bytes()
    if mode.startswith('race') and key.endswith('.tar.gz') and not f.exists():
        winner = b'different winner' if 'different' in mode else data
        f.write_bytes(winner)
        pathlib.Path(str(f)+'.meta').write_text(json.dumps({'sha256': hashlib.sha256(winner).hexdigest()}))
        error('ConditionalRequestConflict' if '409' in mode else 'PreconditionFailed')
    if f.exists(): error('412')
    f.write_bytes(data)
    pathlib.Path(str(f)+'.meta').write_text(json.dumps({'sha256': arg('--metadata').split('=',1)[1]} if '--metadata' in args else {}))
    print('{}')
else: raise AssertionError(op)
''')
    fake.chmod(0o755)
    key = 'packages/kit/1.0.0/kit-1.0.0-darwin-arm64.tar.gz'
    registry = base / 'registry'
    tar = registry / key
    tar.parent.mkdir(parents=True)
    tar.write_bytes(b'package bytes')
    digest = hashlib.sha256(tar.read_bytes()).hexdigest()
    pathlib.Path(str(tar)+'.sha256').write_text('sha256:'+digest+'\n')
    provenance = pathlib.Path(str(tar)+'.provenance.json')
    provenance.write_text(json.dumps({'schemaVersion':1, 'repositoryCommit':'first', 'ciRun':'1'}))
    env = dict(os.environ, PATH=str(binpath)+os.pathsep+os.environ['PATH'],
               R2_ENDPOINT='https://invalid.invalid', BUCKET='fake', AWS_EC2_METADATA_DISABLED='true')
    for name in list(env):
        if name.startswith('AWS_') and name != 'AWS_EC2_METADATA_DISABLED': del env[name]
    count = 0
    def run(label, mode='', success=True, setup=None, reuse=None):
        global count
        store = reuse or base / label
        store.mkdir(exist_ok=True)
        if setup: setup(store)
        result = subprocess.run(['bash', str(root/'scripts/publish-packages.sh'), str(registry)],
                                env=dict(env, FAKE_STORE=str(store), FAKE_MODE=mode), capture_output=True, text=True)
        assert (result.returncode == 0) == success, (label, result.returncode, result.stdout, result.stderr)
        if success:
            for suffix in ('', '.sha256', '.provenance.json'): assert (store/(key+suffix)).exists(), (label,suffix)
        count += 1
        print('PASS', label)
        return store
    store = run('absent')
    calls = (store/'calls').read_text()
    puts = [line for line in calls.splitlines() if line.startswith('put-object')]
    assert puts == ['put-object '+key+s for s in ('', '.sha256', '.provenance.json')]
    before = (store/(key+'.provenance.json')).read_bytes()
    provenance.write_text(json.dumps({'schemaVersion':1, 'repositoryCommit':'second', 'ciRun':'2'}))
    (store/'calls').write_text('')
    run('identical', reuse=store)
    assert 'put-object' not in (store/'calls').read_text()
    assert (store/(key+'.provenance.json')).read_bytes() == before
    def existing(store, metadata=None, suffix='', data=b'package bytes'):
        f = store/(key+suffix)
        f.parent.mkdir(parents=True, exist_ok=True)
        f.write_bytes(data)
        pathlib.Path(str(f)+'.meta').write_text(json.dumps(metadata if metadata is not None else {'sha256':hashlib.sha256(data).hexdigest()}))
    run('mismatch', success=False, setup=lambda s: existing(s, data=b'changed'))
    run('missing-metadata', success=False, setup=lambda s: existing(s, {}))
    for mode in ('403','timeout','misleading404'):
        s = run(mode, mode, False)
        assert 'put-object' not in (s/'calls').read_text()
    run('nosuchkey', 'nosuchkey')
    run('interrupt', 'interrupt', False)
    for mode in ('race412', 'race409', 'race412different', 'race409different'):
        run(mode, mode, 'different' not in mode)
    run('corrupt-checksum', success=False, setup=lambda s: existing(s, suffix='.sha256', data=b'wrong'))
    run('corrupt-provenance', success=False, setup=lambda s: existing(s, suffix='.provenance.json', data=b'{}'))
    # Metadata alone cannot disguise a corrupt checksum sidecar.
    checksum = pathlib.Path(str(tar)+'.sha256').read_bytes()
    run('lying-checksum-metadata', success=False, setup=lambda s: existing(s, {'sha256':hashlib.sha256(checksum).hexdigest()}, '.sha256', b'wrong'))
    # Both publishers observe absence before competing for conditional creation.
    store = base/'concurrent'
    store.mkdir()
    publishers = [subprocess.Popen(['bash', str(root/'scripts/publish-packages.sh'), str(registry)],
        env=dict(env, FAKE_STORE=str(store), FAKE_MODE='concurrent', PUBLISHER=str(i)),
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True) for i in range(2)]
    for publisher in publishers:
        stdout, stderr = publisher.communicate(timeout=20)
        assert publisher.returncode == 0, (stdout, stderr)
    assert (store/key).read_bytes() == tar.read_bytes()
    assert (store/'calls').read_text().count('put-object '+key+'\n') == 2
    print('PASS concurrent publishers both observed absence')
    # Execute the actual workflow shell blocks with fake Git and AWS, offline.
    workflow = (root/'.github/workflows/publish-catalog.yml').read_text()
    def block(name):
        step = workflow.split('      - name: '+name+'\n', 1)[1].split('      - ', 1)[0]
        return '\n'.join(line[10:] for line in step.split('        run: |\n', 1)[1].splitlines())
    git = binpath/'git'
    # Each workflow fetch observes main afresh. Keep the counter outside the
    # subprocess so the second eligibility check can see a newer revision.
    git.write_text("""#!/bin/sh
case "$*" in
  'fetch --no-tags origin main')
    count=0
    [ ! -f "$FAKE_STORE/fetch-count" ] || count=$(cat "$FAKE_STORE/fetch-count")
    echo $((count+1)) > "$FAKE_STORE/fetch-count";;
  'rev-parse HEAD') echo built;;
  'rev-parse FETCH_HEAD')
    if [ "${FAKE_MAIN:-built}" = advances ] && [ "$(cat "$FAKE_STORE/fetch-count")" -gt 1 ]; then
      echo newer
    elif [ "${FAKE_MAIN:-built}" = advances ]; then
      echo built
    else
      echo "${FAKE_MAIN:-built}"
    fi;;
  *) exit 99;;
esac
""")
    git.chmod(0o755)
    index = block('Publish discovery index (mutable; no-cache)')
    eligibility = block('Check current main after acquiring publication job')
    catalog = registry/'catalog'
    catalog.mkdir()
    (catalog/'index.json').write_text('{"schemaVersion":2}')
    (catalog/'index.json.sha256').write_text('index-checksum')
    def workflow_run(label, mode='', main='built', active='true'):
        store = base/label
        store.mkdir()
        script = ("set -euo pipefail\n" + eligibility + "\n" +
                  'guard=$(bash "$REPO/scripts/check-catalog-publication.sh" registry)\n' +
                  '[ "$guard" = publish=true ] || exit 0\n' +
                  'bash "$REPO/scripts/publish-packages.sh" registry\n' + index)
        result = subprocess.run(['bash','-c',script], cwd=base,
            env=dict(env, REPO=str(root), FAKE_STORE=str(store), FAKE_MODE=mode,
                     FAKE_MAIN=main, DIRECTORY_PACKAGES_ENABLED=active, GITHUB_OUTPUT=str(base/'outputs')),
            capture_output=True, text=True)
        return store, result
    store, result = workflow_run('ordered-index')
    assert result.returncode == 0, result.stderr
    puts = [line for line in (store/'calls').read_text().splitlines() if line.startswith('put-object')]
    assert puts == ['put-object '+key+s for s in ('', '.sha256', '.provenance.json')] + [
        'put-object catalog/index.json', 'put-object catalog/index.json.sha256'], puts
    for mode in ('interrupt', 'race412different', '403'):
        store, result = workflow_run('index-blocked-'+mode, mode)
        assert result.returncode != 0
        assert 'put-object catalog/' not in (store/'calls').read_text()
    store, result = workflow_run('stale-index', main='newer')
    assert result.returncode == 0 and '::notice::Stale ref' in result.stdout
    assert 'put-object catalog/' not in (store/'calls').read_text()
    (base/'outputs').write_text('')
    store, result = workflow_run('main-advances', main='advances')
    assert result.returncode == 0 and '::notice::Stale ref' in result.stdout, (result.stdout, result.stderr)
    assert 'index=true' in (base/'outputs').read_text(), 'first check was not eligible'
    assert (store/'fetch-count').read_text().strip() == '2', 'both main checks must execute'
    puts = [line for line in (store/'calls').read_text().splitlines() if line.startswith('put-object')]
    assert puts == ['put-object '+key+s for s in ('', '.sha256', '.provenance.json')], puts
    print('PASS main advances between eligibility checks; immutable packages retained, index not uploaded')
    store, result = workflow_run('inactive-schema', active='false')
    assert result.returncode == 0 and not (store/'calls').exists()
    assert "if: needs.build.outputs.publish == 'true'" in workflow
    assert 'cancel-in-progress: false' in workflow
    build_job = workflow.split('  build:',1)[1].split('  publish:',1)[0]
    assert 'secrets.' not in build_job and 'environment: production' not in build_job
    # The build job gates ITS OWN registry: catalog gate, then activation guard,
    # then upload, all on the same gate-produced registry directory. Bound the
    # job at the top-level `publish:` key, not the `outputs.publish` line.
    build_job = workflow.split('\n  build:\n',1)[1].split('\n  publish:\n',1)[0]
    assert 'secrets.' not in build_job and 'environment: production' not in build_job
    gate = build_job.find('bash scripts/tests/catalog-contract.sh --out "$RUNNER_TEMP/catalog-gate"')
    guard = build_job.find('bash scripts/check-catalog-publication.sh "$RUNNER_TEMP/catalog-gate/registry"')
    upload = build_job.find('path: ${{ runner.temp }}/catalog-gate/registry/')
    assert -1 < gate < guard < upload, (gate, guard, upload)
    assert build_job.count('patronus build') == 0, 'build job must upload only the catalog-gate registry'
    print(str(count)+' publication contract cases and 7 workflow ordering/eligibility cases passed')
PY
