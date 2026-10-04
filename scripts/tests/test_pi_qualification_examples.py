"""C independent-example consistency + M checker consumption; no upstream I proof.

No provisioner, network client, third-party import or fake-action process is started.
"""
import copy
import json
import unittest

import test_pi_evidence as fixtures


class QualificationExampleTests(fixtures.EvidenceFixture):
    def examples(self, name):
        value = json.loads((fixtures.ROOT / 'scripts/qualification/pi-native/evidence/examples' / name).read_text())
        self.assertTrue(value['example'])
        self.assertEqual('unknown', value['status'])
        self.assertEqual('M', value['class'])
        self.assertEqual('C', value['model_class'])
        self.assertEqual('scripts/check-pi-evidence.py', value['production_subject'])
        self.assertEqual('docs/pi-qualification/operational-tests.md', value['scenario_consumer'])
        self.assertIn('INDEPENDENT EXAMPLE MODEL', ' '.join(value['limitations']))
        return {row['scenario']: row for row in value['observations']}

    def events(self, row):
        self.assertEqual(row['expected'], row['observed'])
        self.assertEqual('pass', row['outcome'])
        self.assertEqual({'event': 'observe', 'value': row['observed']}, row['trace'][-1])
        return [event['event'] for event in row['trace'][:-1]]

    def test_approval_vectors_preserve_mutation_order_exception_and_consumption(self):
        rows = self.examples('approval-cases.json')
        self.assertEqual(['approve', 'mutate', 'recheck'], self.events(rows['mutation']))
        self.assertNotEqual(rows['mutation']['approved_argv'], rows['mutation']['observed_argv'])
        self.assertEqual(['mutate', 'final-policy-check'], self.events(rows['order']))
        self.assertEqual(['handler-error', 'blocking-path'], self.events(rows['exception']))
        self.assertEqual(['approval-cancel', 'no-dispatch'], self.events(rows['cancel']))
        self.assertEqual(['deadline-expired', 'no-dispatch'], self.events(rows['expiry']))
        self.assertEqual(['reserve', 'consume', 'reserve-denied'], self.events(rows['parallel-consumption']))
        consumed = [step['call'] for step in rows['parallel-consumption']['trace'] if step['event'] == 'consume']
        self.assertEqual(['one'], consumed)
        self.assertTrue(all(row['observed'] == 'denied' for row in rows.values()))

    def test_provisioning_vectors_bind_argv_scripts_and_partial_preservation(self):
        rows = self.examples('provisioning-cases.json')
        approved = rows['approved-argv']
        self.assertEqual(approved['approved_argv'], approved['observed_argv'])
        self.assertIn('--ignore-scripts', approved['observed_argv'])
        self.assertEqual(['approval-match', 'fake-dispatch'], self.events(approved))
        changed = rows['changed-argv']
        self.assertNotEqual(changed['approved_argv'], changed['observed_argv'])
        self.assertEqual(['argv-changed', 'no-dispatch'], self.events(changed))
        self.assertEqual(['transitive-script-detected', 'scripts-policy', 'no-dispatch'], self.events(rows['scripts-deny']))
        self.assertEqual('deny', rows['scripts-deny']['trace'][1]['value'])
        self.assertEqual(['fake-dispatch', 'interrupt', 'descendant-settlement', 'preserve-outputs'], self.events(rows['interruption']))
        self.assertEqual('unknown', rows['interruption']['trace'][2]['value'])
        self.assertEqual('retained', rows['interruption']['observed'])
        self.assertEqual(['partial-write', 'report-partial', 'preserve-outputs'], self.events(rows['partial-result']))
        self.assertEqual('partial', rows['partial-result']['observed'])

    def test_web_vectors_explicit_failures_overrides_and_no_fallback(self):
        rows = self.examples('web-cases.json')
        expected = {'oversized': 'response-byte-limit', 'redirect': 'redirect-denied',
                    'private-destination': 'private-route-denied', 'challenge': 'challenge-observed',
                    '429': 'rate-limit-observed', 'offline': 'network-unavailable', 'cancel': 'cancel-request',
                    'summary-override': 'summary-route-denied', 'background-fetch': 'background-route-denied',
                    'provider-mismatch': 'provider-revalidation', 'model-mismatch': 'model-revalidation'}
        self.assertEqual(set(expected), set(rows))
        for name, event in expected.items():
            self.assertEqual([event, 'no-fallback'], self.events(rows[name]))
            self.assertEqual('denied', rows[name]['observed'])

    def test_checker_consumes_all_example_observations_only_as_M(self):
        self.assert_qualifies(self.packet('core-exact-pin'))
        for case_id in ('Q-T3-M', 'Q-T5-M', 'Q-T7-M'):
            changed = self.packet('core-exact-pin')
            next(row for row in changed['evidence'] if row['id'] == case_id)['class'] = 'I'
            self.assert_refuses(changed, 'wrong evidence class')

    def test_checker_refuses_false_pass_with_unknown_bypass_failed_model_outcomes(self):
        record = self.packet('core-exact-pin')
        for case_id in ('Q-T3-M', 'Q-T5-M', 'Q-T7-M'):
            for result in ('unknown', 'bypass', 'failed'):
                with self.subTest(case=case_id, result=result):
                    changed = copy.deepcopy(record)
                    case = next(row for row in changed['evidence'] if row['id'] == case_id)
                    case['model_observations'][0]['outcome'] = result
                    self.assert_refuses(changed, 'model outcome')

    def test_checker_refuses_missing_trace_changed_observation_and_missing_negative(self):
        for mutation in ('trace', 'observed', 'missing-negative'):
            changed = self.packet('core-exact-pin')
            case = next(row for row in changed['evidence'] if row['id'] == 'Q-T3-M')
            if mutation == 'trace':
                case['model_observations'][0]['trace'] = []
            elif mutation == 'observed':
                case['model_observations'][0]['observed'] = 'allowed'
            else:
                case['model_observations'].pop()
            self.assert_refuses(changed)


if __name__ == '__main__':
    unittest.main()
