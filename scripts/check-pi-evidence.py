#!/usr/bin/env python3
"""Read-only, offline record completeness checks; never permission or security proof."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat

VERSION = 1
MAX_JSON = 4 * 1024 * 1024
MAX_FILE = 64 * 1024 * 1024
MAX_TOTAL = 2 * 1024 * 1024 * 1024
MAX_READS = 4096
STATUSES = {'verified', 'observed', 'inferred', 'proposed', 'unknown'}
RESULTS = {'pass', 'failed', 'unknown', 'bypass', 'skipped'}
DOSSIER_FIELDS = {
    'identity', 'version_ref', 'archive', 'installed_comparison', 'license_attribution',
    'dependency_tree_sbom', 'lifecycle_build_scripts', 'provenance_signature',
    'reachable_advisories', 'upstream_tests_ci', 'platform_runtime',
    'privilege_network_credentials', 'update_remove_rollback', 'alternatives',
    'residual_risks', 'approval', 'delta_to_latest',
}
CONTRACT_PATH = Path(__file__).resolve().parents[1] / 'scripts/qualification/pi-native/evidence/cases.json'


class Invalid(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise Invalid(message)


def obj(value, fields=()):
    require(isinstance(value, dict), 'expected object')
    require(set(fields) <= value.keys(), 'missing fields: ' + ', '.join(sorted(set(fields) - value.keys())))
    return value


def text(value):
    require(isinstance(value, str) and bool(value.strip()), 'expected nonempty text')
    return value


def known_text(value):
    require(text(value).strip().lower() != 'unknown', 'unknown required identity/observation')
    return value


def sequence(value):
    require(isinstance(value, list), 'expected array')
    return value


def sha(value):
    require(isinstance(value, str) and re.fullmatch('[0-9a-f]{64}', value), 'invalid SHA-256')
    return value


def relative(value, allow_root=False):
    text(value)
    path = PurePosixPath(value)
    require(not path.is_absolute() and '\\' not in value and '\x00' not in value,
            'absolute/invalid input path: ' + value)
    require('..' not in path.parts and (allow_root or value != '.'), 'path traversal/root input: ' + value)
    require(str(path) == value, 'noncanonical relative path: ' + value)
    return value


def timestamp(value):
    text(value)
    require(value.endswith('Z'), 'timestamp must be UTC Z')
    try:
        return datetime.fromisoformat(value[:-1] + '+00:00')
    except ValueError as error:
        raise Invalid('invalid UTC timestamp') from error


def no_duplicate_keys(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, 'duplicate JSON key: ' + key)
        result[key] = value
    return result


def decode(data):
    try:
        value = json.loads(data, object_pairs_hook=no_duplicate_keys,
                           parse_constant=lambda value: (_ for _ in ()).throw(Invalid('nonfinite JSON number')))
        return obj(value)
    except (UnicodeError, json.JSONDecodeError, RecursionError) as error:
        raise Invalid('malformed JSON') from error


class Reader:
    """No symlink components; fstat and bounded reads also reject FIFOs/devices."""
    def __init__(self, root):
        self.root = Path(root).absolute()
        require(self.root.is_dir() and not self.root.is_symlink(), 'root must be a regular directory')
        self.checked = {}
        self.total = 0
        self.reads = 0

    def read(self, name, retain=True):
        relative(name)
        self.reads += 1
        require(self.reads <= MAX_READS, 'read-count limit exceeded')
        # dir_fd + O_NOFOLLOW prevents a checked component being swapped for a symlink.
        fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
        try:
            parts = PurePosixPath(name).parts
            for part in parts[:-1]:
                child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=fd)
                os.close(fd)
                fd = child
            file_fd = os.open(parts[-1], os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=fd)
            with os.fdopen(file_fd, 'rb') as stream:
                info = os.fstat(stream.fileno())
                require(stat.S_ISREG(info.st_mode), 'nonregular input: ' + name)
                limit = MAX_JSON if retain else MAX_FILE
                require(info.st_size <= limit, 'oversized input: ' + name)
                chunks = []
                digest = hashlib.sha256()
                size = 0
                while True:
                    chunk = stream.read(64 * 1024)
                    if not chunk:
                        break
                    size += len(chunk)
                    self.total += len(chunk)
                    require(size <= limit, 'oversized input: ' + name)
                    require(self.total <= MAX_TOTAL, 'total read limit exceeded')
                    digest.update(chunk)
                    if retain:
                        chunks.append(chunk)
        finally:
            os.close(fd)
        data = b''.join(chunks) if retain else None
        actual = digest.hexdigest()
        require(name not in self.checked or self.checked[name] == actual, 'input changed during check: ' + name)
        self.checked[name] = actual
        return data

    def ref(self, value, retain=False):
        obj(value, ('path', 'sha256'))
        expected = sha(value['sha256'])
        data = self.read(value['path'], retain=retain)
        require(self.checked[value['path']] == expected, 'digest mismatch: ' + value['path'])
        return data

    def refs(self, values):
        paths = set()
        for value in sequence(values):
            self.ref(value)
            require(value['path'] not in paths, 'duplicate input path: ' + value['path'])
            paths.add(value['path'])


def ref_map(values):
    return {value['path']: value['sha256'] for value in values}


def same_refs(left, right):
    return ref_map(left) == ref_map(right)


def invocation(value):
    obj(value, ('started_at', 'ended_at'))
    require(timestamp(value['started_at']) <= timestamp(value['ended_at']), 'reversed invocation timestamps')
    require(('argv' in value) != ('native_tool' in value), 'declare exactly one command or native-tool call')
    if 'argv' in value:
        require(bool(sequence(value['argv'])), 'empty argv')
        for arg in value['argv']:
            text(arg)
    else:
        obj(value['native_tool'], ('name', 'arguments'))
        text(value['native_tool']['name'])
        obj(value['native_tool']['arguments'])


def owner(value, claim=None):
    obj(value, ('owner', 'result', 'claim', 'stage', 'inputs'))
    text(value['owner'])
    text(value['stage'])
    require(value['result'] in RESULTS, 'invalid owner result')
    if value['result'] == 'pass':
        known_text(value['owner'])
        known_text(value['stage'])
    if claim is not None:
        require(value['claim'] == claim, 'owner decision claim mismatch')


def grant(value, reader, complete=False, record=None):
    obj(value, ('version', 'kind', 'status', 'issuer', 'issued_at', 'action', 'roots',
                'outputs', 'inputs', 'expires_at', 'revalidation_trigger', 'owner_verification'))
    require(value['version'] == VERSION and type(value['version']) is int, 'unsupported grant version')
    require(value['kind'] == 'grant' and value['status'] in STATUSES, 'invalid grant type/status')
    if 'example' in value:
        require(type(value['example']) is bool, 'invalid grant example label')
    text(value['issuer'])
    text(value['action'])
    issued = timestamp(value['issued_at'])
    roots = sequence(value['roots'])
    require(bool(roots), 'missing grant roots')
    for root in roots:
        relative(root, allow_root=True)
    for output in sequence(value['outputs']):
        relative(output)
    reader.refs(value['inputs'])
    expires = timestamp(value['expires_at']) if value['expires_at'] is not None else None
    if value['revalidation_trigger'] is not None:
        text(value['revalidation_trigger'])
    require(expires is not None or bool(value['revalidation_trigger']), 'missing expiry/revalidation trigger')
    if expires:
        require(issued < expires, 'grant expires before issue')
    verification = obj(value['owner_verification'], ('owner', 'result', 'checked_at', 'action', 'roots', 'outputs', 'inputs'))
    text(verification['owner'])
    checked_at = timestamp(verification['checked_at'])
    require(checked_at >= issued, 'verification predates grant')
    require(verification['result'] in RESULTS, 'invalid approval result')
    reader.refs(verification['inputs'])
    for field in ('action', 'roots', 'outputs', 'inputs'):
        require(verification[field] == value[field], 'changed grant scope: ' + field)
    if complete:
        require(not value.get('example', False), 'example grant cannot qualify')
        known_text(value['issuer'])
        known_text(verification['owner'])
        require(value['status'] in {'verified', 'observed'} and verification['result'] == 'pass', 'missing approved grant')
        require(expires is None or datetime.now(timezone.utc) <= expires, 'declared grant expired')
        require(same_refs(value['inputs'], record['inputs']), 'grant input scope mismatch')
        require(value['action'] == record['owner_acceptance']['stage'], 'grant stage mismatch')
        require(set(value['outputs']) == set(ref_map(record['outputs'])), 'grant output scope mismatch')
        require(issued <= timestamp(record['invocation']['started_at']), 'invocation precedes grant')
        if expires:
            require(timestamp(record['invocation']['ended_at']) <= expires and checked_at <= expires,
                    'invocation/verification after grant expiry')
        for ref in record['inputs'] + record['outputs']:
            require(any(root == '.' or ref['path'].startswith(root + '/') for root in roots),
                    'out-of-scope required input: ' + ref['path'])


def finding_rows(review):
    canonical = {}
    sources = {}
    for finding in sequence(review['findings']):
        obj(finding, ('id', 'sources', 'severity', 'rationale', 'consequence', 'unresolved', 'owner_acceptance'))
        text(finding['id'])
        text(finding['rationale'])
        text(finding['consequence'])
        require(finding['severity'] in {'Critical', 'Major', 'Medium', 'Low'}, 'invalid normalized severity')
        require(type(finding['unresolved']) is bool, 'invalid unresolved state')
        acceptance = obj(finding['owner_acceptance'], ('owner', 'result', 'reason'))
        text(acceptance['owner'])
        text(acceptance['reason'])
        require(acceptance['result'] in RESULTS, 'invalid residual acceptance')
        if acceptance['result'] == 'pass':
            known_text(acceptance['owner'])
            known_text(acceptance['reason'])
        identity = {k: v for k, v in finding.items() if k != 'sources'}
        require(finding['id'] not in canonical or canonical[finding['id']] == identity,
                'inconsistent duplicate canonical finding')
        canonical[finding['id']] = identity
        require(bool(sequence(finding['sources'])), 'missing source finding IDs')
        for source in finding['sources']:
            obj(source, ('id', 'original_severity'))
            text(source['id'])
            text(source['original_severity'])
            meaning = (finding['id'], source['original_severity'])
            require(source['id'] not in sources or sources[source['id']] == meaning,
                    'inconsistent duplicate source finding')
            sources[source['id']] = meaning
    return list(canonical.values())


def review_record(value, reader, complete=False, targets=()):
    obj(value, ('cycles', 'reviewer', 'reviewed', 'corrections', 'findings', 'disposition'))
    require(type(value['cycles']) is int and 0 <= value['cycles'] <= 2, 'review cycle ceiling exceeded')
    text(value['reviewer'])
    reader.refs(value['reviewed'])
    require(value['disposition'] in {'complete', 'incomplete'}, 'invalid review disposition')
    covered = ref_map(value['reviewed'])
    for correction in sequence(value['corrections']):
        obj(correction, ('old', 'new', 'disposition', 'parent', 'verified'))
        # Old bytes are retained separately; no reviewer coverage is attributed to new bytes.
        reader.ref(correction['old'])
        reader.ref(correction['new'])
        known_text(correction['parent'])
        require(correction['disposition'] == 'parent-verified-not-independently-reviewed', 'invalid correction disposition')
        require(correction['old']['sha256'] in covered.values(), 'correction old hash was not reviewed')
        require(correction['old']['sha256'] != correction['new']['sha256'], 'correction hashes unchanged')
        require(correction['verified'] == correction['new'], 'stale parent correction verification')
        covered[correction['new']['path']] = correction['new']['sha256']
    findings = finding_rows(value)
    unresolved = [f for f in findings if f['unresolved']]
    blocked = any(f['severity'] in {'Critical', 'Major'} for f in unresolved)
    mediums = [f for f in unresolved if f['severity'] == 'Medium']
    blocked = blocked or len(mediums) > 2 or any(f['owner_acceptance']['result'] != 'pass' for f in mediums)
    if value['disposition'] == 'complete':
        require(not blocked and value['cycles'] >= 1, 'inconsistent review threshold disposition')
    if complete:
        require(value['disposition'] == 'complete', 'review incomplete')
        for ref in targets:
            require(covered.get(ref['path']) == ref['sha256'], 'stale/missing reviewed hash: ' + ref['path'])


def common(record, reader):
    obj(record, ('version', 'kind', 'example', 'status', 'claim', 'class', 'source', 'inputs', 'outputs',
                 'environment', 'grant', 'invocation', 'result', 'expected', 'observation', 'logs',
                 'exit_code', 'settlement', 'review', 'gates', 'limitations', 'owner_acceptance'))
    require(type(record['version']) is int and record['version'] == VERSION, 'unsupported record version')
    require(type(record['example']) is bool, 'invalid example label')
    require(record['status'] in STATUSES and record['class'] in {'M', 'C', 'I'}, 'invalid status/class')
    source = obj(record['source'], ('root', 'revision', 'dirty_inventory'))
    require(Path(text(source['root'])).is_absolute(), 'source root must be absolute')
    text(source['revision'])
    reader.refs(source['dirty_inventory'])
    for field in ('inputs', 'outputs', 'logs'):
        reader.refs(record[field])
    env = obj(record['environment'], ('platform', 'pins'))
    text(env['platform'])
    require(bool(obj(env['pins'])), 'missing environment pins')
    for name, pin in env['pins'].items():
        text(name)
        text(pin)
    grant(record['grant'], reader)
    invocation(record['invocation'])
    require(record['result'] in RESULTS, 'invalid result')
    text(record['expected'])
    text(record['observation'])
    require(record['exit_code'] is None or type(record['exit_code']) is int, 'invalid exit code')
    require(record['settlement'] in {'settled', 'not-applicable', 'unknown', 'failed'}, 'invalid settlement')
    review_record(record['review'], reader)
    gates = obj(record['gates'], ('required_tests', 'authority', 'release'))
    require(all(value in RESULTS for value in gates.values()), 'invalid gate outcome')
    for limitation in sequence(record['limitations']):
        text(limitation)
    owner(record['owner_acceptance'], record['claim'])
    reader.refs(record['owner_acceptance']['inputs'])


def candidate(record, reader, contract, complete=False):
    common(record, reader)
    require(record['claim'] in contract['claims'], 'unknown claim ID')
    obj(record, ('identity', 'digest', 'retrieved_at', 'immutable_refs', 'dossier'))
    text(record['identity'])
    if record['digest'] is not None:
        sha(record['digest'])
    timestamp(record['retrieved_at'])
    for ref in sequence(record['immutable_refs']):
        obj(ref, ('url', 'ref'))
        text(ref['url'])
        text(ref['ref'])
    dossier = obj(record['dossier'], DOSSIER_FIELDS)
    for field in DOSSIER_FIELDS:
        evidence = obj(dossier[field], ('status', 'value', 'references'))
        require(evidence['status'] in STATUSES, 'invalid dossier evidence status: ' + field)
        reader.refs(evidence['references'])
        if complete:
            require(evidence['status'] in {'verified', 'observed'} and evidence['value'] not in (None, '', 'unknown'),
                    'unknown/inferred candidate field: ' + field)
            require(bool(evidence['references']), 'missing candidate evidence: ' + field)
    if complete:
        known_text(record['identity'])
        require(not record['example'] and record['digest'] is not None and bool(record['immutable_refs']),
                'candidate identity/digest/immutable refs incomplete')
        require(record['digest'] in ref_map(dossier['archive']['references']).values(),
                'candidate digest does not identify archive bytes')
        for field in dossier.values():
            require(all(ref_map(record['inputs'] + record['outputs']).get(ref['path']) == ref['sha256']
                        for ref in field['references']), 'candidate evidence outside reviewed inputs/outputs')
        complete_common(record, reader)


def complete_common(record, reader):
    require(not record['example'] and record['status'] in {'observed', 'verified'}, 'example/unknown record cannot qualify')
    require(record['result'] == 'pass' and record['exit_code'] == 0, 'failed/unknown required test')
    require(record['settlement'] in {'settled', 'not-applicable'}, 'unsettled work')
    for value in (record['expected'], record['observation'], record['review']['reviewer']):
        known_text(value)
    require(bool(record['inputs']) and bool(record['outputs']) and bool(record['logs']), 'missing inputs/outputs/logs')
    require(re.fullmatch('[0-9a-f]{40}|[0-9a-f]{64}', record['source']['revision']), 'missing exact source revision')
    require(record['environment']['platform'] != 'unknown' and
            all(pin != 'unknown' for pin in record['environment']['pins'].values()), 'unknown environment/pins')
    require(all(value == 'pass' for value in record['gates'].values()), 'separate acceptance gate blocked')
    require(record['owner_acceptance']['result'] == 'pass' and
            same_refs(record['owner_acceptance']['inputs'], record['inputs']), 'missing/mismatched owner acceptance')
    grant(record['grant'], reader, complete=True, record=record)
    require(all(ref_map(record['inputs']).get(ref['path']) == ref['sha256']
                for ref in record['source']['dirty_inventory']), 'dirty inventory outside granted inputs')
    targets = record['inputs'] + record['outputs'] + record['logs'] + record['source']['dirty_inventory']
    require(set(ref_map(record['logs'])) <= set(ref_map(record['outputs'])), 'logs not declared outputs')
    review_record(record['review'], reader, complete=True, targets=targets)


def required_sets(record, contract, selected, mcp):
    dependencies = record['dependencies']
    groups = {group for name in selected for group in contract['claims'][name]['dependency_groups']}
    roles = {role for group in groups for role in contract['dependency_roles'][group]}
    require(roles <= {dep['role'] for dep in dependencies}, 'missing selected dependency role')

    def expand(name):
        rule = contract['claims'][name]
        required = set(rule['required'])
        for parent in rule['parents']:
            require(parent in selected, 'missing selected lower-level claim: ' + parent)
            required.update(expand(parent))
        for dep in dependencies:
            if dep['group'] in rule['dependency_groups']:
                required.update(selector + '@' + dep['role'] + ':' + dep['digest']
                                for selector in contract['dependency_selectors'])
        if name != 'static-delivery' and mcp:
            required.update(contract['runtime_mcp_selectors'])
        return required

    return {name: expand(name) for name in selected}


def runtime_applicability(record, reader, selected):
    if selected == {'static-delivery'}:
        return False
    value = obj(record['applicability'], ('mcp-enabled', 'effective_config', 'inventory',
                                          'owner', 'result', 'interpretation', 'reviewed'))
    require(type(value['mcp-enabled']) is bool, 'missing/unknown MCP applicability')
    known_text(value['owner'])
    known_text(value['interpretation'])
    require(value['result'] == 'pass', 'unreviewed applicability')
    reader.ref(value['effective_config'])
    inventory = decode(reader.ref(value['inventory'], retain=True))
    obj(inventory, ('selected_mcp_surfaces',))
    surfaces = sequence(inventory['selected_mcp_surfaces'])
    for surface in surfaces:
        text(surface)
    require(bool(surfaces) == value['mcp-enabled'], 'conflicting MCP inventory')
    reader.refs(value['reviewed'])
    refs = [value['effective_config'], value['inventory']]
    require(same_refs(value['reviewed'], refs), 'stale applicability reviewed hashes')
    require(all(ref_map(record['inputs']).get(ref['path']) == ref['sha256'] for ref in refs),
            'applicability hashes not bound to inputs')
    require('code-intel-exact-pin' not in selected or value['mcp-enabled'], 'code-intel requires MCP')
    return value['mcp-enabled']


def model_observations(case, rule):
    """Check reported example consistency, not upstream policy behavior."""
    rows = sequence(case.get('model_observations'))
    scenarios = set()
    for row in rows:
        obj(row, ('scenario', 'trace', 'approved_argv', 'observed_argv', 'expected', 'observed', 'outcome'))
        name = text(row['scenario'])
        require(name in rule['outcomes'] and name not in scenarios, 'unknown/duplicate model scenario')
        scenarios.add(name)
        for field in ('approved_argv', 'observed_argv'):
            for argument in sequence(row[field]):
                text(argument)
        trace = sequence(row['trace'])
        require(bool(trace), 'missing model trace')
        for step in trace:
            obj(step, ('event',))
            text(step['event'])
        text(row['expected'])
        text(row['observed'])
        require(trace[-1] == {'event': 'observe', 'value': row['observed']}, 'model trace observation mismatch')
        require(row['outcome'] in RESULTS and row['outcome'] == case['outcomes'][name],
                'inconsistent model outcome')
        if row['outcome'] == 'pass':
            require(row['expected'] == row['observed'] and row['observed'] not in {'unknown', 'failed', 'bypass'},
                    'claimed pass with unknown/bypass/failed model outcome')
    require(scenarios == set(rule['outcomes']), 'missing mandatory model scenario')


def case_record(case, record, reader, contract):
    obj(case, ('id', 'class', 'dependency', 'inputs', 'result', 'outcomes', 'invocation',
               'logs', 'exit_code', 'settlement', 'expected', 'observation'))
    require(case['id'] in contract['cases'], 'unknown case ID: ' + str(case['id']))
    rule = contract['cases'][case['id']]
    require(case['class'] in rule['classes'], 'wrong evidence class: ' + case['id'])
    reader.refs(case['inputs'])
    require(same_refs(case['inputs'], record['inputs']), 'case input/dependency update mismatch')
    invocation(case['invocation'])
    require(timestamp(record['invocation']['started_at']) <= timestamp(case['invocation']['started_at']) and
            timestamp(case['invocation']['ended_at']) <= timestamp(record['invocation']['ended_at']),
            'case outside declared invocation interval')
    reader.refs(case['logs'])
    require(all(ref_map(record['outputs']).get(ref['path']) == ref['sha256'] for ref in case['logs']),
            'case logs not bound to outputs')
    text(case['expected'])
    text(case['observation'])
    require(type(case['exit_code']) is int or case['exit_code'] is None, 'invalid case exit code')
    require(case['settlement'] in {'settled', 'not-applicable', 'failed', 'unknown'}, 'invalid case settlement')
    require(case['result'] in RESULTS, 'invalid case result')
    outcomes = obj(case['outcomes'])
    require(set(outcomes) == set(rule['outcomes']), 'missing/unknown mandatory outcomes: ' + case['id'])
    require(all(value in RESULTS for value in outcomes.values()), 'invalid mandatory outcome')
    if case['result'] == 'pass':
        known_text(case['expected'])
        known_text(case['observation'])
        require(all(value == 'pass' for value in outcomes.values()), 'claimed pass with unknown/bypass/failed mandatory outcome')
        require(case['exit_code'] == 0 and bool(case['logs']), 'pass lacks successful exit/logs')
        require(case['settlement'] == 'settled' if case['class'] == 'I' else
                case['settlement'] in {'settled', 'not-applicable'}, 'pass lacks settlement')
    if 'limit' in rule:
        model_observations(case, rule)
    selector = case['id'] + ':' + case['class']
    if rule['scope'] == 'dependency':
        dep = obj(case['dependency'], ('role', 'identity', 'digest'))
        sha(dep['digest'])
        require(any(all(dep[key] == item[key] for key in dep) for item in record['dependencies']),
                'dependency-scoped evidence identity/digest mismatch')
        selector += '@' + text(dep['role']) + ':' + dep['digest']
    else:
        require(case['dependency'] is None, 'unexpected dependency scope')
    if case['id'] == 'OP-WEB':
        bounds = obj(case.get('bounds'))
        require(set(bounds) == set(rule['bounds']) and all(result in RESULTS for result in bounds.values()),
                'web requires explicit unresolved bounds')
    return selector


def validate_selector(selector, contract):
    base, separator, dependency = selector.partition('@')
    parts = base.split(':')
    require(len(parts) == 2 and parts[0] in contract['cases'], 'unknown required case ID')
    rule = contract['cases'][parts[0]]
    require(parts[1] in rule['classes'], 'wrong required case class')
    if rule['scope'] == 'dependency':
        role, colon, digest = dependency.partition(':')
        require(separator and colon and re.fullmatch('[a-z0-9][a-z0-9-]*', role), 'missing/invalid required dependency scope')
        sha(digest)
    else:
        require(not separator, 'unexpected required dependency scope')


def qualification(record, reader, contract, contract_digest, requested):
    common(record, reader)
    obj(record, ('cases', 'claims', 'dependencies', 'applicability', 'deployment_acceptance', 'evidence'))
    require(record['claim'] in contract['claims'], 'unknown claim ID')
    reader.ref(record['cases'])
    require(record['cases']['sha256'] == contract_digest, 'cases contract differs from checker contract')
    if 'required_set_rules' in record:
        require(record['required_set_rules'] == contract['claims'], 'template required rules differ from contract')
    claims = obj(record['claims'])
    require(set(claims) == set(contract['claims']), 'missing/unknown claim IDs; optional claims must be explicitly unselected')
    selected = set()
    for name, value in claims.items():
        obj(value, ('status', 'required', 'decision'))
        require(value['status'] in {'selected', 'unselected'}, 'invalid claim selection')
        required = sequence(value['required'])
        require(all(isinstance(item, str) for item in required) and len(required) == len(set(required)), 'invalid/duplicate required selectors')
        for selector in required:
            validate_selector(selector, contract)
        owner(value['decision'], name)
        reader.refs(value['decision']['inputs'])
        if value['status'] == 'selected':
            selected.add(name)
        else:
            require(not required, 'unselected claim has required set')
    roles = set()
    candidates = []
    for dep in sequence(record['dependencies']):
        obj(dep, ('role', 'group', 'identity', 'digest', 'candidate'))
        require(re.fullmatch('[a-z0-9][a-z0-9-]*', text(dep['role'])), 'invalid dependency role')
        text(dep['identity'])
        sha(dep['digest'])
        require(dep['role'] not in roles, 'duplicate dependency role')
        roles.add(dep['role'])
        require(dep['group'] in contract['dependency_roles'], 'unknown dependency group')
        data = decode(reader.ref(dep['candidate'], retain=True))
        require(data.get('kind') == 'candidate', 'dependency is not candidate record')
        candidate(data, reader, contract)
        require(dep['identity'] == data['identity'] and dep['digest'] == data['digest'], 'candidate identity/digest mismatch')
        require(ref_map(record['inputs']).get(dep['candidate']['path']) == dep['candidate']['sha256'], 'candidate not bound to inputs')
        candidates.append((dep, data))
    indexed = {}
    for case in sequence(record['evidence']):
        selector = case_record(case, record, reader, contract)
        require(selector not in indexed, 'duplicate case selector: ' + selector)
        indexed[selector] = case
    # Record mode preserves honestly incomplete packets; it still refuses false pass rows.
    if requested is None:
        return
    require(requested in contract['claims'] and requested == record['claim'], 'requested claim mismatch/unknown claim')
    require(requested in selected, 'requested claim is unselected')
    require(record['class'] == 'I' if requested != 'static-delivery' else record['class'] in {'M', 'C'},
            'qualification class mismatch')
    complete_common(record, reader)
    mcp = runtime_applicability(record, reader, selected)
    requirements = required_sets(record, contract, selected, mcp)
    groups = {group for name in selected for group in contract['claims'][name]['dependency_groups']}
    for dep, data in candidates:
        require(dep['group'] in groups, 'dependency selected outside claim scope')
        require(dep['role'] not in {role for group, roles in contract['dependency_roles'].items()
                                   if group != dep['group'] for role in roles}, 'dependency role in wrong group')
        candidate(data, reader, contract, complete=True)
    for name in selected:
        value = claims[name]
        require(set(value['required']) == requirements[name], 'reduced/changed required set: ' + name)
        decision = value['decision']
        require(decision['result'] == 'pass' and same_refs(decision['inputs'], record['inputs']) and
                decision['stage'] == record['owner_acceptance']['stage'], 'missing/mismatched claim owner decision: ' + name)
        for selector in requirements[name]:
            require(selector in indexed and indexed[selector]['result'] == 'pass', 'missing/failed required case: ' + selector)
    if 'unattended-mutation' in selected:
        decision = record['deployment_acceptance']
        owner(decision, requested)
        reader.refs(decision['inputs'])
        require(decision['result'] == 'pass' and decision['stage'] == 'whole-process-deployment' and
                same_refs(decision['inputs'], record['inputs']), 'missing whole-process deployment acceptance')
    if 'web-bounded-unattended' in selected:
        require(all(result == 'pass' for result in indexed['OP-WEB:I']['bounds'].values()), 'missing hard web enforcement')


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('record', 'qualification'))
    parser.add_argument('--root', required=True)
    parser.add_argument('--file', required=True)
    parser.add_argument('--claim')
    args = parser.parse_args(argv)
    reader = None
    try:
        reader = Reader(args.root)
        record = decode(reader.read(args.file))
        contract_reader = Reader(CONTRACT_PATH.parent)
        contract_bytes = contract_reader.read(CONTRACT_PATH.name)
        contract = decode(contract_bytes)
        contract_digest = hashlib.sha256(contract_bytes).hexdigest()
        if record.get('kind') == 'grant':
            require(args.mode == 'record', 'grant is not qualification evidence')
            grant(record, reader)
        elif record.get('kind') == 'candidate':
            require(args.mode == 'record', 'candidate is not a qualification packet')
            candidate(record, reader, contract)
        elif record.get('kind') == 'qualification':
            qualification(record, reader, contract, contract_digest,
                          args.claim if args.mode == 'qualification' else None)
        else:
            raise Invalid('unsupported record kind')
        require(args.mode != 'qualification' or args.claim is not None, 'qualification requires --claim')
        result = {'ok': True, 'mode': args.mode, 'claim': args.claim, 'checked': reader.checked,
                  'contract_sha256': contract_digest,
                  'limit': 'Record consistency only; no authority, authenticity or security certification.'}
        code = 0
    except (Invalid, OSError, KeyError, TypeError, ValueError, RecursionError) as error:
        result = {'ok': False, 'mode': args.mode, 'claim': args.claim,
                  'checked': reader.checked if reader else {}, 'errors': [str(error)]}
        code = 1
    print(json.dumps(result, sort_keys=True))
    return code


if __name__ == '__main__':
    raise SystemExit(main())
