#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# ///
"""Sync one upstream release into origin/main.

Called from .github/workflows/sync-upstream-release.yaml. Reads config from
environment variables (see main() below); does not accept CLI args. Runs
inside a checkout of the internal fork with the internal repo as origin.

The flow:
    1. Look up upstream and internal latest release tags via `gh`.
    2. If equal, exit 0.
    3. Merge the upstream tag into main. Resolve modify/delete conflicts
       under .github/workflows/ by keeping the delete (`git rm`); resolve
       modify/delete on files owned by this fork by keeping our version
       (`git checkout --ours`). Any other unresolved state aborts.
    4. Regenerate mta.yaml from the merged upstream template.
    5. Disable renovate on the internal repo.
    6. Remove any upstream workflows that don't target our solinas runners.
    7. Push main and create the internal release matching the upstream tag.
"""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

# Paths this fork owns; the merge must never take upstream's version.
# The `ours` merge driver is registered on the workflow's runner via
# `git config merge.ours.driver true`; `.gitattributes` maps these
# patterns to it. This list is a safety net for modify/delete
# conflicts, which merge drivers do not cover.
FORK_OWNED_WORKFLOW_FILES = {
    ".github/workflows/piper.yaml",
    ".github/workflows/sync-upstream-release.yaml",
    ".github/workflows/codeql-sast-analysis.yml",
    ".github/workflows/secret-scanning-sast-analysis.yml",
}

# Marker used by the "keep only workflows targeting internal runners" step.
SOLINAS_MARKER = "runs-on:"  # narrow further by content match below

INTERNAL_HOST = "github.tools.sap"
UPSTREAM_REPO = "cloudfoundry/app-autoscaler"


def git(*args: str, check: bool = True, capture: bool = True) -> str:
    """Run `git <args>` and return stripped stdout.

    Errors surface as CalledProcessError with stderr included; we let them
    propagate so the workflow step fails visibly with the git message.
    """
    result = subprocess.run(
        ["git", *args],
        check=check,
        capture_output=capture,
        text=True,
    )
    return result.stdout.strip() if capture else ""


def gh(*args: str, env_overrides: dict[str, str] | None = None) -> str:
    """Run `gh <args>` with optional env overrides (for switching hosts)."""
    env = os.environ.copy()
    if env_overrides:
        env.update(env_overrides)
    result = subprocess.run(
        ["gh", *args],
        check=True,
        capture_output=True,
        text=True,
        env=env,
    )
    return result.stdout.strip()


def latest_release_tag(repo: str, *, host: str, token: str) -> str:
    """Return the tagName of the latest release, or 'none' if there isn't one."""
    try:
        return gh(
            "release", "view",
            "--repo", repo,
            "--json", "tagName",
            "--jq", ".tagName",
            env_overrides={"GH_HOST": host, "GH_TOKEN": token},
        )
    except subprocess.CalledProcessError:
        return "none"


def configure_git_identity() -> None:
    git("config", "user.email", "dl_5ed0ca884687a4027d3f8880@global.corp.sap")
    git("config", "user.name", "aas-serviceuser")
    # No-op merge driver referenced by .gitattributes; always keeps ours.
    git("config", "merge.ours.driver", "true")


def add_upstream_remote_and_fetch() -> None:
    # `add` fails if the remote already exists (e.g. re-runs on same runner).
    try:
        git("remote", "add", "upstream", f"https://github.com/{UPSTREAM_REPO}.git")
    except subprocess.CalledProcessError:
        pass
    git("fetch", "upstream", "--tags", "--force", capture=False)


def unmerged_paths() -> list[tuple[str, str]]:
    """Return [(status_code, path), ...] for entries with a merge conflict.

    Uses the porcelain v1 status format. Statuses covered:
        AA, DD, DU, UD, UA, AU, UU — every code where either the index or
        the worktree column is 'U' (unmerged) or matches the both-added /
        both-deleted patterns.
    """
    out = git("status", "--porcelain=v1", "-z")
    if not out:
        return []
    entries: list[tuple[str, str]] = []
    for raw in out.split("\0"):
        if len(raw) < 3:
            continue
        code, path = raw[:2], raw[3:]
        if code in {"AA", "DD", "DU", "UD", "UA", "AU", "UU"}:
            entries.append((code, path))
    return entries


def resolve_conflicts() -> None:
    """Auto-resolve modify/delete conflicts on upstream workflow files."""
    for code, path in unmerged_paths():
        if code not in {"DU", "UD"}:
            raise SystemExit(
                f"Unresolved {code} conflict on {path!r} — human intervention required."
            )
        if path in FORK_OWNED_WORKFLOW_FILES:
            # Our fork owns this file; keep our version.
            git("checkout", "--ours", "--", path)
            git("add", "--", path)
        elif path.startswith(".github/workflows/"):
            # Upstream workflow deleted on our fork on purpose; keep the delete.
            git("rm", "-f", "--", path)
        else:
            raise SystemExit(
                f"Unhandled modify/delete conflict on {path!r} — human intervention required."
            )

    # Belt & braces: nothing must remain in an unmerged state.
    remaining = unmerged_paths()
    if remaining:
        raise SystemExit(f"Unresolved conflicts remain: {remaining!r}")


def merge_upstream(tag: str) -> None:
    """Merge upstream tag into current branch, resolving known conflicts."""
    # Capture merge output so git's "Automatic merge failed; fix conflicts
    # and then commit the result." doesn't land in the log on the happy path
    # (we ARE fixing the conflicts and committing, right below). If the
    # resolver can't handle something, we re-print the captured output as
    # part of the failure diagnostics.
    proc = subprocess.run(
        ["git", "merge", tag, "--no-edit",
         "-m", f"chore: merge upstream release {tag}"],
        capture_output=True,
        text=True,
    )
    if proc.returncode == 0:
        return  # clean merge, we're done

    try:
        resolve_conflicts()
    except SystemExit:
        # Surface the git output that we swallowed above so a human can
        # see the CONFLICT lines when they investigate.
        sys.stderr.write(proc.stdout)
        sys.stderr.write(proc.stderr)
        raise
    git("commit", "--no-edit", "-m", f"chore: merge upstream release {tag}")


def regenerate_mta_yaml(tag: str) -> None:
    """Regenerate mta.yaml from the (now-merged) upstream template."""
    version = tag.removeprefix("v")
    subprocess.run(
        ["uv", "run", ".github/scripts/generate-mta-yaml.py", version],
        check=True,
    )
    git("add", "-f", "mta.yaml")
    # Only commit if the regeneration actually changed something.
    if subprocess.run(
        ["git", "diff", "--cached", "--quiet"], capture_output=True
    ).returncode != 0:
        git("commit", "-m", f"chore: generate mta.yaml for release {tag}")


def disable_renovate() -> None:
    Path("renovate.json").write_text(
        '{ "$schema": "https://docs.renovatebot.com/renovate-schema.json", "enabled": false }\n'
    )
    if subprocess.run(
        ["git", "diff", "--quiet", "renovate.json"], capture_output=True
    ).returncode != 0:
        git("commit", "-am", "chore: disable renovate on internal repo")


def remove_non_internal_workflows() -> None:
    """Remove upstream workflow files that don't target our self-hosted runner."""
    workflow_dir = Path(".github/workflows")
    if not workflow_dir.is_dir():
        return
    removed_any = False
    for path in sorted(workflow_dir.glob("*.y*ml")):
        rel = path.as_posix()
        if rel in FORK_OWNED_WORKFLOW_FILES:
            continue
        # Keep any workflow that already targets our self-hosted runner set.
        content = path.read_text(errors="replace")
        if "self-hosted" in content and "solinas" in content and SOLINAS_MARKER in content:
            continue
        git("rm", "-f", "--", rel)
        removed_any = True
    if removed_any:
        git("commit", "-m", "chore: remove upstream workflows not targeting internal runners")


def push_main() -> None:
    git("push", "origin", "main", capture=False)


def create_internal_release(tag: str, token: str) -> None:
    subprocess.run(
        ["gh", "auth", "login", "--hostname", INTERNAL_HOST, "--with-token"],
        input=token,
        text=True,
        check=True,
    )
    subprocess.run(
        [
            "gh", "release", "create", tag,
            "--title", tag,
            "--notes",
            f"Synced from upstream {UPSTREAM_REPO} release {tag}.",
            "--target", "main",
        ],
        check=True,
        env={**os.environ, "GH_HOST": INTERNAL_HOST, "GH_TOKEN": token},
    )


def main() -> int:
    internal_repo = os.environ["INTERNAL_REPO"]  # e.g. autoscaler/app-autoscaler
    internal_token = os.environ["INTERNAL_TOKEN"]
    upstream_com_token = os.environ["UPSTREAM_COM_TOKEN"]

    upstream_tag = latest_release_tag(
        UPSTREAM_REPO, host="github.com", token=upstream_com_token,
    )
    internal_tag = latest_release_tag(
        internal_repo, host=INTERNAL_HOST, token=internal_token,
    )
    print(f"upstream_tag={upstream_tag}")
    print(f"internal_tag={internal_tag}")

    if upstream_tag == internal_tag:
        print(f"Already at upstream {upstream_tag}, nothing to do.")
        return 0

    configure_git_identity()
    add_upstream_remote_and_fetch()

    merge_upstream(upstream_tag)
    regenerate_mta_yaml(upstream_tag)
    disable_renovate()
    remove_non_internal_workflows()
    push_main()
    create_internal_release(upstream_tag, internal_token)
    return 0


if __name__ == "__main__":
    sys.exit(main())
