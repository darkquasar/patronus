"""Offline checks of allowed and rejected publication attempts."""
import copy
import importlib.util
from pathlib import Path
import subprocess
import unittest
from unittest.mock import Mock

SPEC = importlib.util.spec_from_file_location(
    "publication", Path(__file__).resolve().parents[1] / "check-publication-authorization.py"
)
publication = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(publication)
SHA = "a" * 40
HEAD = "b" * 40
OWNER = {"id": publication.OWNER_ID, "login": "darkquasar"}
OTHER = {"id": 123, "login": "contributor"}


class PublicationAuthorizationTests(unittest.TestCase):
    def setUp(self):
        self.env = {
            "GITHUB_REPOSITORY": publication.REPOSITORY,
            "GITHUB_SHA": SHA,
            "GITHUB_EVENT_NAME": "push",
            "GITHUB_REF": "refs/heads/main",
            "GITHUB_ACTOR": "darkquasar",
            "GITHUB_ACTOR_ID": str(publication.OWNER_ID),
            "GITHUB_TRIGGERING_ACTOR": "darkquasar",
        }
        self.pr = {
            "number": 7, "merged": True, "merge_commit_sha": SHA,
            "base": {"ref": "main", "repo": {"full_name": publication.REPOSITORY}},
            "head": {"sha": HEAD, "repo": {"full_name": "contributor/fork"}},
            "user": OTHER, "merged_by": OWNER,
            "merged_at": "2026-10-23T06:00:00Z",
        }
        self.reviews = [{"user": OWNER, "state": "APPROVED", "commit_id": HEAD,
                         "submitted_at": "2026-10-23T05:00:00Z"}]
        self.get = Mock(side_effect=self.api)
        self.run = Mock()

    def api(self, path, paginate=False):
        if "/commits/" in path:
            self.assertTrue(paginate)
            return [{"number": self.pr["number"]}]
        if "/reviews?" in path:
            self.assertTrue(paginate)
            return self.reviews
        self.assertFalse(paginate)
        return self.pr

    def check(self, mode="catalog"):
        return publication.authorize(mode, self.env, self.get, self.run)

    def test_owner_approved_fork_pr_after_merge_is_allowed(self):
        self.assertIn("#7", self.check())

    def test_manual_main_run_revalidates_approval(self):
        self.env["GITHUB_EVENT_NAME"] = "workflow_dispatch"
        self.assertIn("#7", self.check())
        self.assertTrue(self.get.called)

    def test_pr_fork_branch_tag_and_indirect_events_are_denied_before_api(self):
        for changes in [
            {"GITHUB_REPOSITORY": "contributor/patronus"},
            {"GITHUB_EVENT_NAME": "pull_request", "GITHUB_REF": "refs/pull/7/merge"},
            {"GITHUB_EVENT_NAME": "pull_request_target"},
            {"GITHUB_EVENT_NAME": "workflow_run"},
            {"GITHUB_REF": "refs/heads/feature"},
            {"GITHUB_REF": "refs/tags/v1.0.0"},
            {"GITHUB_EVENT_NAME": "workflow_dispatch", "GITHUB_REF": "refs/heads/feature"},
        ]:
            with self.subTest(changes=changes):
                env = {**self.env, **changes}
                with self.assertRaises(ValueError):
                    publication.authorize("catalog", env, self.get, self.run)
        self.get.assert_not_called()

    def test_direct_push_and_wrong_merge_are_denied(self):
        original = copy.deepcopy(self.pr)
        for changes in [
            {"merged": False}, {"merge_commit_sha": "c" * 40},
            {"base": {"ref": "feature", "repo": {"full_name": publication.REPOSITORY}}},
            {"base": {"ref": "main", "repo": {"full_name": "contributor/fork"}}},
        ]:
            with self.subTest(changes=changes):
                self.pr = {**original, **changes}
                with self.assertRaises(ValueError):
                    self.check()
        self.get.side_effect = lambda *args: []
        with self.assertRaises(ValueError):
            self.check()

    def test_owner_pr_requires_owner_to_merge(self):
        self.pr["user"] = OWNER
        self.reviews = []
        self.assertIn("#7", self.check())
        for merger in [OTHER, None]:
            self.pr["merged_by"] = merger
            with self.assertRaises(ValueError):
                self.check()

    def test_nonowner_review_or_spoofed_login_is_denied(self):
        for user in [OTHER, {"id": 123, "login": "darkquasar"}]:
            self.reviews[0]["user"] = user
            with self.assertRaises(ValueError):
                self.check()

    def test_absent_stale_dismissed_or_negative_approval_is_denied(self):
        for reviews in [
            [], [{"user": OWNER, "state": "APPROVED", "commit_id": "c" * 40}],
            [{"user": OWNER, "state": "DISMISSED", "commit_id": HEAD}],
            self.reviews + [{"user": OWNER, "state": "CHANGES_REQUESTED", "commit_id": HEAD}],
        ]:
            with self.subTest(reviews=reviews):
                self.reviews = reviews
                with self.assertRaises(ValueError):
                    self.check()

    def test_comment_does_not_revoke_approval(self):
        self.reviews.append({"user": OWNER, "state": "COMMENTED", "commit_id": HEAD})
        self.assertIn("#7", self.check())

    def test_approval_added_only_after_merge_is_denied(self):
        self.reviews[0]["submitted_at"] = "2026-10-23T07:00:00Z"
        with self.assertRaises(ValueError):
            self.check()

    def test_api_failure_denies_publication(self):
        self.get.side_effect = subprocess.CalledProcessError(1, "gh")
        with self.assertRaises(subprocess.CalledProcessError):
            self.check()

    def test_owner_release_checks_tag_ancestry(self):
        self.env["GITHUB_REF"] = "refs/tags/v1.0.0"
        self.assertIn("release", self.check("release"))
        self.assertEqual(self.run.call_args_list[-1].args[0],
                         ["git", "merge-base", "--is-ancestor", SHA, "FETCH_HEAD"])
        self.get.assert_not_called()

    def test_nonowner_release_fork_and_rerun_are_denied_before_git(self):
        self.env["GITHUB_REF"] = "refs/tags/v1.0.0"
        for changes in [
            {"GITHUB_ACTOR": "contributor", "GITHUB_ACTOR_ID": "123"},
            {"GITHUB_ACTOR_ID": "123"}, {"GITHUB_TRIGGERING_ACTOR": "contributor"},
            {"GITHUB_REPOSITORY": "contributor/patronus"},
            {"GITHUB_EVENT_NAME": "workflow_dispatch"}, {"GITHUB_REF": "refs/heads/main"},
        ]:
            with self.subTest(changes=changes):
                with self.assertRaises(ValueError):
                    publication.authorize("release", {**self.env, **changes}, self.get, self.run)
        self.run.assert_not_called()

    def test_tag_off_main_is_denied(self):
        self.env["GITHUB_REF"] = "refs/tags/v1.0.0"
        self.run.side_effect = [None, subprocess.CalledProcessError(1, "git")]
        with self.assertRaises(subprocess.CalledProcessError):
            self.check("release")


if __name__ == "__main__":
    unittest.main()
