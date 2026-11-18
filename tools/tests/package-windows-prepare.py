"""Render two synthetic apps with the accepted T3 code; never publish fixtures."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--tap-root', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    sys.path.insert(0, str(args.tap_root.resolve()))
    from distribution.contracts import OWNER_ID, canonical_digest
    from distribution.snapshot import freeze, validate_snapshot
    from distribution.validation import render, validate_candidate
    import yaml

    out = args.out.resolve()
    out.mkdir(parents=True)  # Refuse to mix this evidence with another run.
    assets = out / 'assets'
    assets.mkdir()
    source = out / 'main.go'
    source.write_text('package main\nimport "fmt"\nvar name, version string\n'
                      'func main() { fmt.Println(name + " " + version) }\n')
    environment = dict(os.environ, CGO_ENABLED='0', GOOS='windows', GO111MODULE='off',
                       GOPROXY='off', GOSUMDB='off', GOTOOLCHAIN='local')
    records = []
    fixtures = args.tap_root / 'distribution/tests/fixtures'
    for app in ['patronus', 'lumen']:
        registration = json.loads((fixtures / f'{app}-native-registration.json').read_bytes())
        identity = copy.deepcopy(json.loads((fixtures / f'{app}-snapshot.json').read_bytes())
                                 ['payload']['identity'])
        for platform, asset in registration['configuration']['assets'].items():
            path = assets / asset
            if platform.startswith('windows-'):
                env = dict(environment, GOARCH=platform.split('-')[1])
                command = ['go', 'build', '-trimpath', '-ldflags',
                           f'-s -w -X main.name={app} -X main.version={identity["version"]}',
                           '-o', str(path), str(source)]
                result = subprocess.run(command, env=env, capture_output=True, text=True, timeout=180)
                records.append(dict(command=command, exit=result.returncode,
                                    stdout=result.stdout, stderr=result.stderr))
                (out / 'builds.json').write_text(json.dumps(records, indent=2) + '\n')
                result.check_returncode()
            else:
                path.write_bytes(('synthetic:' + asset).encode())
            data = path.read_bytes()
            identity['assets'][asset] = dict(size=len(data), sha256=hashlib.sha256(data).hexdigest())
        checksums = ''.join(item['sha256'] + '  ' + name + '\n'
                            for name, item in sorted(identity['assets'].items())).encode()
        (out / f'{app}-checksums.txt').write_bytes(checksums)
        identity['checksums_sha256'] = hashlib.sha256(checksums).hexdigest()
        snapshot = freeze(registration, identity, dict(kind='baseline', owner_id=OWNER_ID,
            reference='fixture-only:native-windows', identity_digest=canonical_digest(identity)))
        validate_snapshot(registration, snapshot)
        (out / f'{app}-snapshot.json').write_text(json.dumps(snapshot, indent=2) + '\n')
        (out / f'{app}-registration.json').write_text(json.dumps(registration, indent=2) + '\n')
        for channel in ['scoop', 'winget']:
            candidate = render(channel, registration, snapshot)
            assert candidate == render(channel, registration, snapshot)
            validation = validate_candidate(channel, registration, snapshot, candidate)
            (out / f'{app}-{channel}-validation.json').write_text(json.dumps(validation, indent=2) + '\n')
            if not validation['passed']:
                raise RuntimeError(validation)
            corrupted = dict(candidate)
            first = next(iter(corrupted))
            corrupted[first] += b'\n'  # Canonical-byte identity must fail closed.
            denied = validate_candidate(channel, registration, snapshot, corrupted)
            (out / f'{app}-{channel}-negative.json').write_text(json.dumps(denied, indent=2) + '\n')
            if denied['passed'] or denied['failure_code'] != 'validation-failed':
                raise RuntimeError('Changed candidate was not rejected', denied)
            for relative, data in candidate.items():
                path = out / 'canonical' / channel / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(data)
                # Local transport changes only. Canonical identity validation above
                # never admits these loopback URLs as publishable candidates.
                if channel == 'scoop':
                    document = json.loads(data)
                    for entry in document['architecture'].values():
                        entry['url'] = 'http://127.0.0.1:18765/' + entry['url'].rsplit('/', 1)[1]
                    adapted = (json.dumps(document, indent=2) + '\n').encode()
                else:
                    document = yaml.safe_load(data)
                    for entry in document.get('Installers', []):
                        entry['InstallerUrl'] = ('http://127.0.0.1:18765/'
                                                 + entry['InstallerUrl'].rsplit('/', 1)[1])
                    header = data.splitlines(keepends=True)[0] if data.startswith(b'# yaml-language-server: ') else b''
                    adapted = header + yaml.safe_dump(document, allow_unicode=True, sort_keys=False, width=1000).encode()
                path = out / 'local-transport' / channel / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(adapted)
    inventory = {str(path.relative_to(out)): hashlib.sha256(path.read_bytes()).hexdigest()
                 for path in sorted(out.rglob('*')) if path.is_file()}
    (out / 'sha256.json').write_text(json.dumps(inventory, indent=2) + '\n')
    print('Prepared two synthetic apps; canonical candidates validated; local transport explicitly separate.')


if __name__ == '__main__':
    main()
