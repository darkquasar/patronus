#!/usr/bin/env python3
"""Fail closed before catalogue credentials or release write permissions are used."""
import json
import os
import re
import subprocess
import sys
from datetime import datetime

REPOSITORY = "darkquasar/patronus"
OWNER_ID = 28435534


def api(path, paginate=False):
    command = ["gh", "api", "-H", "Accept: application/vnd.github+json"]
    if paginate:
        command += ["--paginate", "--slurp"]
    result = json.loads(subprocess.check_output(command + [path], text=True))
    return [item for page in result for item in page] if paginate else result


def is_owner(user):
    return user.get("id") == OWNER_ID


def approved_pr(pr, sha, reviews):
    """Review the exact merged change, never just another PR containing its commit."""
    if not (
        pr.get("merged") is True
        and pr.get("merge_commit_sha") == sha
        and pr.get("base", {}).get("ref") == "main"
        and pr.get("base", {}).get("repo", {}).get("full_name") == REPOSITORY
    ):
        return False
    if is_owner(pr.get("user", {})):
        # GitHub prohibits self-review; only the owner's own merge substitutes.
        return is_owner(pr.get("merged_by") or {})
    decisions = [
        review for review in reviews
        if is_owner(review.get("user", {}))
        and review.get("state") in {"APPROVED", "CHANGES_REQUESTED", "DISMISSED"}
    ]
    # GitHub returns reviews in chronological order. Comments do not revoke an
    # approval; new commits, requested changes and dismissed approvals do.
    return bool(
        decisions
        and decisions[-1]["state"] == "APPROVED"
        and decisions[-1].get("commit_id") == pr.get("head", {}).get("sha")
        and decisions[-1].get("commit_id")
        and datetime.fromisoformat(decisions[-1]["submitted_at"].replace("Z", "+00:00"))
        <= datetime.fromisoformat(pr["merged_at"].replace("Z", "+00:00"))
    )


def authorize(mode, env, get=api, run=subprocess.run):
    if env.get("GITHUB_REPOSITORY") != REPOSITORY:
        raise ValueError("publication is restricted to " + REPOSITORY)
    sha = env.get("GITHUB_SHA", "")
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        raise ValueError("a full GitHub commit SHA is required")
    event, ref = env.get("GITHUB_EVENT_NAME"), env.get("GITHUB_REF", "")
    if mode == "catalog":
        if event not in {"push", "workflow_dispatch"} or ref != "refs/heads/main":
            raise ValueError("catalogue publication requires a main push or main manual run")
        associated = get(f"repos/{REPOSITORY}/commits/{sha}/pulls?per_page=100", True)
        for summary in associated:
            number = summary["number"]
            pr = get(f"repos/{REPOSITORY}/pulls/{number}")
            reviews = get(f"repos/{REPOSITORY}/pulls/{number}/reviews?per_page=100", True)
            if approved_pr(pr, sha, reviews):
                return f"Authorized catalogue from owner-approved merged PR #{number}."
        raise ValueError("commit must be the result of an owner-approved PR merged into main")
    if mode != "release":
        raise ValueError("expected catalog or release")
    if not (
        event == "push"
        and ref.startswith("refs/tags/v")
        and env.get("GITHUB_ACTOR_ID") == str(OWNER_ID)
        and env.get("GITHUB_ACTOR") == "darkquasar"
        and env.get("GITHUB_TRIGGERING_ACTOR") == "darkquasar"
    ):
        raise ValueError("only darkquasar may push or rerun a v* release tag")
    run(["git", "fetch", "--no-tags", "origin", "main"], check=True)
    run(["git", "merge-base", "--is-ancestor", sha, "FETCH_HEAD"], check=True)
    return "Authorized owner-created release tag on main history."


if __name__ == "__main__":
    try:
        if len(sys.argv) != 2:
            raise ValueError("usage: check-publication-authorization.py catalog|release")
        print(authorize(sys.argv[1], os.environ))
    except (ValueError, KeyError, TypeError, subprocess.CalledProcessError) as exc:
        sys.exit(f"Publication authorization denied: {exc}")
