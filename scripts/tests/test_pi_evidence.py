"""M tests: invented evidence bytes, never upstream execution or release proof."""
import contextlib
import copy
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('pi_evidence', ROOT / 'scripts/check-pi-evidence.py')
CHECKER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CHECKER)


def digest(data):
    return hashlib.sha256(data).hexdigest()


class EvidenceFixture(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def write_json(self, path, value):
        data = (json.dumps(value, indent=2) + '\n').encode()
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)
        return {'path': path, 'sha256': digest(data)}

    def check(self, value, mode='record', claim=None):
        self.write_json('record.json', value)
        args = [mode, '--root', str(self.root), '--file', 'record.json']
        if claim:
            args += ['--claim', claim]
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            status = CHECKER.main(args)
        result = json.loads(output.getvalue())
        self.assertEqual(status == 0, result['ok'], result)
        return status, result

    def common_fixture(self):
        record = json.loads((ROOT / 'docs/pi-qualification/templates/candidate.json').read_text())
        input_ref = self.write_json('inputs/invented.json', {'fixture': 'not upstream bytes'})
        log_ref = self.write_json('outputs/log.json', {'fixture': 'observed fake actions only'})
        record.update(example=False, status='observed', result='pass', exit_code=0,
                      settlement='not-applicable', inputs=[input_ref], outputs=[log_ref], logs=[log_ref],
                      expected='invented expected observation', observation='invented observed result')
        record['source']['revision'] = 'a' * 40
        record['environment'] = {'platform': 'invented-os/invented-cpu', 'pins': {'fake-runtime': '1.2.3'}}
        record['gates'] = dict.fromkeys(record['gates'], 'pass')
        record['review'].update(cycles=1, reviewer='independent-fixture-reviewer',
                                reviewed=[input_ref, log_ref], disposition='complete')
        record['owner_acceptance'].update(owner='fixture-owner', result='pass', inputs=record['inputs'])
        record['grant'].update(example=False, status='observed', issuer='fixture-owner',
                               inputs=record['inputs'], outputs=['outputs/log.json'])
        record['grant']['owner_verification'].update(owner='fixture-owner', result='pass',
                                                     inputs=record['inputs'], outputs=['outputs/log.json'])
        return record

    def packet(self, claim='static-delivery', mcp=False):
        contract = json.loads((ROOT / 'docs/pi-qualification/cases.json').read_text())
        record = self.common_fixture()
        for key in ('identity', 'digest', 'retrieved_at', 'immutable_refs', 'dossier'):
            del record[key]
        record.update(kind='qualification', claim=claim, **{'class': 'C' if claim == 'static-delivery' else 'I'})
        contract_bytes = (ROOT / 'docs/pi-qualification/cases.json').read_bytes()
        (self.root / 'cases.json').write_bytes(contract_bytes)
        record['cases'] = {'path': 'cases.json', 'sha256': digest(contract_bytes)}
        selected = set()
        def select(name):
            selected.add(name)
            for parent in contract['claims'][name]['parents']:
                select(parent)
        select(claim)
        groups = {group for name in selected for group in contract['claims'][name]['dependency_groups']}
        record['dependencies'] = []
        for group in sorted(groups):
            for role in contract['dependency_roles'][group]:
                candidate = self.common_fixture()
                artifact = self.write_json('inputs/' + role + '-artifact.json', {'invented-package': role})
                candidate['inputs'].append(artifact)
                candidate['review']['reviewed'].append(artifact)
                candidate.update(identity='invented-' + role, digest=artifact['sha256'],
                                 immutable_refs=[{'url': 'https://example.invalid/' + role, 'ref': 'f' * 40}])
                for field in candidate['dossier'].values():
                    field.update(status='observed', value='invented evidence, not real qualification', references=candidate['inputs'])
                ref = self.write_json('candidates/' + role + '.json', candidate)
                record['dependencies'].append({'role': role, 'group': group, 'identity': candidate['identity'],
                                               'digest': candidate['digest'], 'candidate': ref})
                record['inputs'].append(ref)
        record['applicability'] = None
        if claim != 'static-delivery':
            config = self.write_json('inputs/effective-config.json', {'fake': 'config', 'mcp': mcp})
            inventory = self.write_json('inputs/inventory.json', {'selected_mcp_surfaces': ['fake-mcp'] if mcp else []})
            record['inputs'] += [config, inventory]
            record['applicability'] = {'mcp-enabled': mcp, 'effective_config': config, 'inventory': inventory,
                                       'owner': 'fixture-owner', 'result': 'pass',
                                       'interpretation': 'invented selected surfaces' if mcp else 'no MCP surface selected',
                                       'reviewed': [config, inventory]}
        def required(name):
            rule = contract['claims'][name]
            result = set(rule['required'])
            for parent in rule['parents']:
                result.update(required(parent))
            for dep in record['dependencies']:
                if dep['group'] in rule['dependency_groups']:
                    result.update(selector + '@' + dep['role'] + ':' + dep['digest']
                                  for selector in contract['dependency_selectors'])
            if name != 'static-delivery' and mcp:
                result.update(contract['runtime_mcp_selectors'])
            return result
        record['claims'] = {}
        for name in contract['claims']:
            record['claims'][name] = {'status': 'selected' if name in selected else 'unselected',
                'required': sorted(required(name)) if name in selected else [],
                'decision': {'owner': 'fixture-owner', 'result': 'pass', 'claim': name,
                             'stage': 'qualification', 'inputs': record['inputs']}}
        record['evidence'] = []
        for selector in sorted(required(claim)):
            base, _, scoped = selector.partition('@')
            case_id, cls = base.split(':')
            dep = next((d for d in record['dependencies'] if scoped == d['role'] + ':' + d['digest']), None)
            case = {'id': case_id, 'class': cls, 'dependency':
                    {k: dep[k] for k in ('role', 'identity', 'digest')} if dep else None,
                    'inputs': record['inputs'], 'result': 'pass',
                    'outcomes': dict.fromkeys(contract['cases'][case_id]['outcomes'], 'pass'),
                    'invocation': record['invocation'], 'logs': record['logs'], 'exit_code': 0,
                    'settlement': 'settled' if cls == 'I' else 'not-applicable',
                    'expected': 'fake expected observation', 'observation': 'fake matching observation'}
            examples = {'Q-T3-M': 'approval-cases.json', 'Q-T5-M': 'provisioning-cases.json',
                        'Q-T7-M': 'web-cases.json'}
            if case_id in examples:
                case['model_observations'] = json.loads((ROOT / 'docs/pi-qualification/examples' /
                                                        examples[case_id]).read_text())['observations']
            if case_id == 'OP-WEB':
                case['bounds'] = dict.fromkeys(contract['cases']['OP-WEB']['bounds'],
                                             'pass' if claim == 'web-bounded-unattended' else 'unknown')
            record['evidence'].append(case)
        record['review']['reviewed'] = record['inputs'] + record['outputs']
        record['grant']['inputs'] = record['inputs']
        record['grant']['owner_verification']['inputs'] = record['inputs']
        record['owner_acceptance'].update(claim=claim, inputs=record['inputs'])
        record['deployment_acceptance'] = {'owner': 'fixture-owner', 'result': 'pass',
            'claim': claim, 'stage': 'whole-process-deployment', 'inputs': record['inputs']} if 'unattended-mutation' in selected else None
        return record

    def assert_qualifies(self, record):
        status, result = self.check(record, 'qualification', record['claim'])
        self.assertEqual(0, status, result)

    def assert_refuses(self, record, reason=None, mode='qualification'):
        status, result = self.check(record, mode, record['claim'] if mode == 'qualification' else None)
        self.assertEqual(1, status, result)
        if reason:
            self.assertIn(reason, ' '.join(result['errors']))
        return result


class EvidenceTests(EvidenceFixture):
    def test_static_delivery_without_runtime_applicability_or_I_passes(self):
        record = self.packet()
        self.assertIsNone(record['applicability'])
        self.assertFalse(any(case['class'] == 'I' for case in record['evidence']))
        self.assert_qualifies(record)

    def test_candidate_digest_must_name_actual_archive_bytes(self):
        record = self.packet('core-exact-pin')
        dep = record['dependencies'][0]
        candidate = json.loads((self.root / dep['candidate']['path']).read_text())
        old = dep['candidate']['sha256']
        candidate['digest'] = 'd' * 64
        new = self.write_json(dep['candidate']['path'], candidate)
        record = json.loads(json.dumps(record).replace(old, new['sha256']).replace(dep['digest'], 'd' * 64))
        self.assert_refuses(record, 'archive bytes')

    def test_core_with_mandatory_web_and_no_mcp_passes(self):
        record = self.packet('core-exact-pin')
        self.assertIn('pi-web-access', {dep['role'] for dep in record['dependencies']})
        self.assertFalse(any(case['id'].endswith('-MCP') for case in record['evidence']))
        self.assert_qualifies(record)

    def test_core_mandatory_web_dossier_and_runtime_cannot_be_omitted(self):
        for missing in ('pi-web-access', 'OP-WEB', 'Q-T7-M'):
            with self.subTest(missing=missing):
                record = self.packet('core-exact-pin')
                record['evidence'] = [case for case in record['evidence'] if case['id'] != missing and
                    (case['dependency'] or {}).get('role') != missing]
                self.assert_refuses(record, 'missing/failed required case')

    def test_static_delivered_mcp_configuration_is_not_runtime_selection(self):
        record = self.packet()
        config = self.write_json('inputs/static-mcp.json', {'servers': {'invented': 'inert bytes'}})
        record['inputs'].append(config)
        record['review']['reviewed'].append(config)
        self.assert_qualifies(record)

    def test_mcp_fixed_union_requires_both_subcases(self):
        record = self.packet('core-exact-pin', mcp=True)
        self.assert_qualifies(record)
        for missing in ('OP-TRUST-MCP', 'OP-APPROVAL-MCP'):
            changed = copy.deepcopy(record)
            changed['evidence'] = [case for case in changed['evidence'] if case['id'] != missing]
            self.assert_refuses(changed, 'missing/failed required case')

    def test_code_intel_integration_mapping_requires_actual_I_subcases(self):
        record = self.packet('code-intel-exact-pin', mcp=True)
        self.assert_qualifies(record)
        case = next(case for case in record['evidence'] if case['id'] == 'OP-CODEINTEL')
        self.assertEqual({'I-T1', 'I-T3', 'I-T4', 'I-T6', 'I-T7'}, set(case['outcomes']))
        for outcome in ('unknown', 'failed', 'skipped'):
            changed = copy.deepcopy(record)
            next(c for c in changed['evidence'] if c['id'] == 'OP-CODEINTEL')['outcomes']['I-T3'] = outcome
            self.assert_refuses(changed, 'mandatory outcome')
        case['class'] = 'C'
        self.assert_refuses(record, 'wrong evidence class')
        self.assert_refuses(self.packet('code-intel-exact-pin', mcp=False), 'code-intel requires MCP')

    def test_functional_core_does_not_imply_hard_web_bounds(self):
        self.assert_qualifies(self.packet('web-functional'))
        record = self.packet('web-bounded-unattended')
        self.assert_qualifies(record)
        for result in ('unknown', 'failed', 'bypass', 'skipped'):
            for bound in ('response-bytes', 'aggregate-output', 'call-limit', 'cost-limit', 'route-denial'):
                changed = copy.deepcopy(record)
                next(case for case in changed['evidence'] if case['id'] == 'OP-WEB')['bounds'][bound] = result
                self.assert_refuses(changed, 'hard web enforcement')

    def test_composite_claim_inputs_and_parent_decision_match(self):
        record = self.packet('core-exact-pin')
        record['claims']['static-delivery']['decision']['inputs'] = []
        self.assert_refuses(record, 'claim owner decision')
        record = self.packet('core-exact-pin')
        record['claims']['static-delivery']['status'] = 'unselected'
        record['claims']['static-delivery']['required'] = []
        self.assert_refuses(record, 'lower-level claim')

    def test_applicability_unknown_missing_conflicting_and_stale_refuses(self):
        for mutation in ('unknown', 'missing-inventory', 'false-selected', 'stale-config', 'unreviewed'):
            with self.subTest(mutation=mutation):
                record = self.packet('core-exact-pin')
                app = record['applicability']
                if mutation == 'unknown':
                    app['mcp-enabled'] = 'unknown'
                elif mutation == 'missing-inventory':
                    del app['inventory']
                elif mutation == 'false-selected':
                    ref = self.write_json('inputs/selected.json', {'selected_mcp_surfaces': ['mcp']})
                    app['inventory'] = ref
                elif mutation == 'stale-config':
                    (self.root / app['effective_config']['path']).write_text('changed config')
                else:
                    app['result'] = 'unknown'
                self.assert_refuses(record)

    def test_wrong_class_skipped_failed_or_missing_I_never_passes(self):
        for mutation in ('M', 'C', 'skipped', 'failed', 'missing-negative'):
            record = self.packet('core-exact-pin')
            case = next(c for c in record['evidence'] if c['id'] == 'OP-TRUST-PI')
            if mutation in ('M', 'C'):
                case['class'] = mutation
            elif mutation == 'missing-negative':
                del case['outcomes']['missing-trust']
            else:
                case['result'] = mutation
            self.assert_refuses(record)

    def test_unknown_ids_reduced_contract_and_required_sets_refuse(self):
        for mutation in ('case', 'claim', 'required', 'contract', 'optional-implicit'):
            record = self.packet()
            if mutation == 'case':
                record['evidence'][0]['id'] = 'OP-TRUST'
            elif mutation == 'claim':
                record['claim'] = 'unknown-claim'
            elif mutation == 'required':
                record['claims']['static-delivery']['required'].pop()
            elif mutation == 'optional-implicit':
                del record['claims']['code-intel-exact-pin']
            else:
                contract = json.loads((self.root / 'cases.json').read_text())
                contract['claims']['static-delivery']['required'] = []
                record['cases'] = self.write_json('cases.json', contract)
            self.assert_refuses(record)

    def test_unknown_embedded_candidate_claim_refuses_in_both_modes(self):
        for mode in ('record', 'qualification'):
            with self.subTest(mode=mode):
                record = self.packet('core-exact-pin')
                dep = record['dependencies'][0]
                path = dep['candidate']['path']
                candidate = json.loads((self.root / path).read_text())
                candidate['claim'] = 'UNKNOWN-CLAIM'
                candidate['owner_acceptance']['claim'] = 'UNKNOWN-CLAIM'
                self.assert_refuses(candidate, 'unknown claim ID', mode='record')
                old = dep['candidate']['sha256']
                new = self.write_json(path, candidate)
                record = json.loads(json.dumps(record).replace(old, new['sha256']))
                self.assert_refuses(record, 'unknown claim ID', mode=mode)

    def test_candidate_high_impact_unknowns_block_and_record_mode_retains_them(self):
        for field in ('dependency_tree_sbom', 'lifecycle_build_scripts', 'reachable_advisories', 'delta_to_latest'):
            record = self.packet('core-exact-pin')
            dep = record['dependencies'][0]
            path = dep['candidate']['path']
            candidate = json.loads((self.root / path).read_text())
            candidate['dossier'][field]['status'] = 'unknown'
            status, result = self.check(candidate)
            self.assertEqual(0, status, result)
            new = self.write_json(path, candidate)
            # Refresh declared bytes everywhere, without hiding the unknown field.
            old = dep['candidate']['sha256']
            record = json.loads(json.dumps(record).replace(old, new['sha256']))
            self.assert_refuses(record, 'unknown/inferred candidate field')

    def test_missing_identity_and_dependency_update_refuse(self):
        record = self.packet('core-exact-pin')
        record['dependencies'][0]['identity'] = ''
        self.assert_refuses(record)
        record = self.packet('core-exact-pin')
        record['dependencies'][0]['digest'] = 'b' * 64
        self.assert_refuses(record, 'identity/digest mismatch')
        record = self.packet('core-exact-pin')
        case = next(c for c in record['evidence'] if c['dependency'])
        case['dependency']['digest'] = 'b' * 64
        self.assert_refuses(record, 'identity/digest mismatch')

    def finding(self, number, severity='Medium', accepted=True):
        return {'id': 'finding-' + str(number),
                'sources': [{'id': 'reviewer-a-' + str(number), 'original_severity': 'Important'}],
                'severity': severity, 'rationale': 'fixture consequence normalization',
                'consequence': 'invented bounded defect', 'unresolved': True,
                'owner_acceptance': {'owner': 'fixture-owner', 'result': 'pass' if accepted else 'unknown',
                                     'reason': 'explicit residual decision'}}

    def test_review_deduplicates_zero_one_two_accepted_medium_one_cycle(self):
        for count in (0, 1, 2):
            record = self.packet()
            findings = [self.finding(n) for n in range(count)]
            if findings:
                duplicate = copy.deepcopy(findings[0])
                duplicate['sources'][0]['id'] = 'reviewer-b-same-defect'
                findings.append(duplicate)
            record['review']['findings'] = findings
            self.assert_qualifies(record)
        record['review']['findings'] = [self.finding(1, 'Low', accepted=False)]
        self.assert_qualifies(record)

    def test_review_blockers_inconsistent_duplicates_rationale_and_third_cycle(self):
        for mutation in ('Critical', 'Major', 'three-medium', 'unaccepted', 'inconsistent-id',
                         'inconsistent-source', 'missing-rationale', 'third-cycle'):
            record = self.packet()
            review = record['review']
            review['findings'] = [self.finding(1)]
            if mutation in ('Critical', 'Major'):
                review['findings'][0]['severity'] = mutation
            elif mutation == 'three-medium':
                review['findings'] = [self.finding(n) for n in range(3)]
            elif mutation == 'unaccepted':
                review['findings'] = [self.finding(1, accepted=False)]
            elif mutation == 'inconsistent-id':
                review['findings'].append(self.finding(1, 'Low'))
            elif mutation == 'inconsistent-source':
                duplicate = self.finding(2)
                duplicate['sources'] = review['findings'][0]['sources']
                review['findings'].append(duplicate)
            elif mutation == 'missing-rationale':
                review['findings'][0]['rationale'] = ''
            else:
                review['cycles'] = 3
            self.assert_refuses(record)

    def test_explicit_unknown_identity_or_observation_cannot_be_relabelled_pass(self):
        for field in ('expected', 'observation'):
            record = self.packet()
            record[field] = 'unknown'
            self.assert_refuses(record, 'unknown')
        for section, field in (('review', 'reviewer'), ('owner_acceptance', 'owner'), ('grant', 'issuer')):
            record = self.packet()
            record[section][field] = 'unknown'
            self.assert_refuses(record, 'unknown')

    def test_separate_required_test_authority_release_gates_and_approval(self):
        for field in ('required_tests', 'authority', 'release'):
            record = self.packet()
            record['gates'][field] = 'failed'
            self.assert_refuses(record, 'separate acceptance gate')
        record = self.packet()
        record['grant']['owner_verification']['result'] = 'unknown'
        self.assert_refuses(record, 'approved grant')
        record = self.packet()
        del record['grant']['owner_verification']
        self.assert_refuses(record, 'missing fields')

    def test_expiry_changed_scope_and_out_of_scope_inputs_refuse(self):
        record = self.packet()
        record['grant']['expires_at'] = '2026-01-02T00:00:00Z'
        self.assert_refuses(record, 'expired')
        record = self.packet()
        record['grant']['action'] = 'publication'
        self.assert_refuses(record, 'changed grant scope')
        record = self.packet()
        record['grant']['roots'] = ['unrelated']
        record['grant']['owner_verification']['roots'] = ['unrelated']
        self.assert_refuses(record, 'out-of-scope')

    def test_unattended_requires_whole_process_acceptance_and_settlement(self):
        record = self.packet('unattended-mutation')
        self.assert_qualifies(record)
        record['deployment_acceptance']['result'] = 'unknown'
        self.assert_refuses(record, 'deployment acceptance')
        record = self.packet('unattended-mutation')
        case = next(c for c in record['evidence'] if c['id'] == 'OP-ISOLATION')
        case['settlement'] = 'unknown'
        self.assert_refuses(record, 'settlement')

    def test_stale_review_and_corrected_bytes_do_not_inherit_review(self):
        record = self.packet()
        old = record['outputs'][0]
        original = (self.root / old['path']).read_bytes()
        (self.root / 'outputs/old-log.json').write_bytes(original)
        old_copy = {'path': 'outputs/old-log.json', 'sha256': old['sha256']}
        new = self.write_json(old['path'], {'corrected': 'new invented bytes'})
        # Fresh input/output hash alone cannot invent independent review coverage.
        record['outputs'] = [new]
        record['logs'] = [new]
        for case in record['evidence']:
            case['logs'] = [new]
        self.assert_refuses(record, 'digest mismatch')
        record['review']['reviewed'] = record['inputs'] + [old_copy]
        self.assert_refuses(record, 'stale/missing reviewed hash')
        record['review']['corrections'] = [{'old': old_copy, 'new': new,
            'disposition': 'parent-verified-not-independently-reviewed', 'parent': 'fixture-owner', 'verified': new}]
        self.assert_qualifies(record)
        self.assertNotIn(new, record['review']['reviewed'])
        record['review']['corrections'][0]['verified'] = old_copy
        self.assert_refuses(record, 'stale parent')

    def test_traversal_symlink_oversized_nonregular_and_versions_refuse(self):
        record = self.packet()
        for path in ('../escape', '/absolute', 'inputs/../escape', 'inputs//invented.json'):
            changed = copy.deepcopy(record)
            changed['inputs'][0]['path'] = path
            self.assert_refuses(changed)
        (self.root / 'symlink').symlink_to(self.root / 'inputs/invented.json')
        changed = copy.deepcopy(record)
        changed['inputs'][0]['path'] = 'symlink'
        self.assert_refuses(changed)
        (self.root / 'outside-link').symlink_to('/tmp', target_is_directory=True)
        changed['inputs'][0]['path'] = 'outside-link/no-such-file'
        self.assert_refuses(changed)
        with (self.root / 'large').open('wb') as stream:
            stream.truncate(CHECKER.MAX_FILE + 1)
        changed['inputs'][0]['path'] = 'large'
        self.assert_refuses(changed, 'oversized')
        import os
        os.mkfifo(self.root / 'fifo')
        changed['inputs'][0]['path'] = 'fifo'
        self.assert_refuses(changed, 'nonregular')
        record['version'] = 2
        self.assert_refuses(record, 'unsupported record version')

    def test_ignored_output_mutation_invalidates_with_unchanged_HEAD_and_tracked_diff(self):
        import os
        import subprocess
        record = self.packet()
        env = {**os.environ, 'HOME': str(self.root), 'GIT_CONFIG_NOSYSTEM': '1',
               'PI_CODING_AGENT_DIR': str(self.root / 'pi-fixture')}
        def git(*args):
            return subprocess.check_output(['git', '-c', 'user.name=Fixture', '-c',
                'user.email=fixture@example.invalid', '-c', 'commit.gpgsign=false', *args],
                cwd=self.root, env=env, stderr=subprocess.STDOUT)
        git('init', '-q')
        (self.root / '.gitignore').write_text('outputs/\n')
        git('add', '.gitignore')
        git('commit', '-qm', 'invented fixture baseline')
        git('check-ignore', 'outputs/log.json')
        before = (git('rev-parse', 'HEAD'), git('diff', '--no-ext-diff', 'HEAD'))
        self.assert_qualifies(record)
        changed = b'ignored output changed without any tracked change\n'
        (self.root / 'outputs/log.json').write_bytes(changed)
        self.assertEqual(before, (git('rev-parse', 'HEAD'), git('diff', '--no-ext-diff', 'HEAD')))
        result = self.assert_refuses(record, 'digest mismatch')
        self.assertEqual(digest(changed), result['checked']['outputs/log.json'])

    def test_real_cli_exit_status_json_and_fixture_isolation(self):
        import os
        import subprocess
        import sys
        record = self.packet()
        self.write_json('record.json', record)
        command = [sys.executable, str(ROOT / 'scripts/check-pi-evidence.py'), 'qualification',
                   '--root', str(self.root), '--file', 'record.json', '--claim', 'static-delivery']
        env = {**os.environ, 'PYTHONDONTWRITEBYTECODE': '1',
               'PI_CODING_AGENT_DIR': str(self.root / 'pi-fixture')}
        good = subprocess.run(command, env=env, capture_output=True, text=True, check=False)
        self.assertEqual(0, good.returncode, good.stderr + good.stdout)
        self.assertTrue(json.loads(good.stdout)['ok'])
        (self.root / 'outputs/log.json').write_text('changed bytes')
        bad = subprocess.run(command, env=env, capture_output=True, text=True, check=False)
        self.assertEqual(1, bad.returncode, bad.stderr + bad.stdout)
        self.assertFalse(json.loads(bad.stdout)['ok'])
        self.assertEqual('', bad.stderr)

    def test_malformed_duplicate_keys_nonfinite_and_wrong_types_refuse_without_traceback(self):
        for raw in (b'{broken', b'{"kind":"candidate","kind":"grant"}', b'{"x": NaN}',
                    b'[]', b'null', b'"text"', b'{"kind": []}'):
            (self.root / 'bad.json').write_bytes(raw)
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                code = CHECKER.main(['record', '--root', str(self.root), '--file', 'bad.json'])
            self.assertEqual(1, code)
            self.assertFalse(json.loads(output.getvalue())['ok'])

    def test_contract_dependency_graph_is_acyclic_and_web_is_mandatory_core(self):
        contract = json.loads((ROOT / 'docs/pi-qualification/cases.json').read_text())
        def visit(name, ancestors):
            self.assertNotIn(name, ancestors)
            for parent in contract['claims'][name]['parents']:
                visit(parent, ancestors | {name})
        for name in contract['claims']:
            visit(name, set())
        self.assertIn('C-T8:C', contract['claims']['static-delivery']['required'])
        self.assertIn('pi-web-access', contract['dependency_roles']['core'])
        self.assertIn('OP-WEB:I', contract['claims']['core-exact-pin']['required'])
        self.assertNotIn('web-functional', contract['claims']['core-exact-pin']['parents'])
        self.assertEqual(['I'], contract['cases']['I-T3']['classes'])

    def test_unknown_required_case_and_class_refuse_even_in_record_mode(self):
        for selector in ('UNKNOWN:M', 'D-T1:I'):
            record = self.packet()
            record['claims']['static-delivery']['required'].append(selector)
            self.assert_refuses(record, mode='record')

    def test_honestly_unknown_candidate_is_structurally_valid(self):
        candidate = json.loads((ROOT / 'docs/pi-qualification/templates/candidate.json').read_text())
        status, result = self.check(candidate)
        self.assertEqual(0, status, result)
        self.assertEqual(digest((self.root / 'record.json').read_bytes()), result['checked']['record.json'])


if __name__ == '__main__':
    unittest.main()
