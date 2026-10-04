#!/usr/bin/env python3
"""Immutable restraml discovery. Python 3.11+, standard library, POSIX."""

import argparse
import fcntl
import hashlib
import http.client
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from contextlib import contextmanager
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any

REPOSITORY = "tikoci/restraml"
FORMAT = "routeros-discovery@1"
SHA = re.compile(r"[0-9a-f]{40}")
VERSION = re.compile(
    r"7\.(0|[1-9][0-9]{0,3})(?:\.(0|[1-9][0-9]{0,3})|(beta|rc)([1-9][0-9]{0,3})|_ab([1-9][0-9]{0,9}))?"
)
ARTIFACTS = {
    "openapi.json",
    "inspect.json",
    "deep-inspect.json",
    "deep-inspect.x86.json",
    "deep-inspect.arm64.json",
    "nightly.json",
}
VOLATILE = {"generatedAt", "builtAt", "bootMs", "enrichmentDurationMs"}
STAGES = ("discovered", "generated", "tested", "published")
LIMIT = 64 * 1024 * 1024


class DiscoveryError(Exception):
    pass


class Missing(Exception):
    pass


def check(condition, message):
    if not condition:
        raise DiscoveryError(message)


def version(value):
    check(
        isinstance(value, str) and VERSION.fullmatch(value), "invalid RouterOS version"
    )
    return value


def order(value):
    m = VERSION.fullmatch(version(value))
    assert m is not None  # version() validated the match.
    minor, patch, pre, number, nightly = m.groups()
    return (
        7,
        int(minor),
        0 if nightly else 1 if pre == "beta" else 2 if pre == "rc" else 3,
        int(nightly or number or patch or 0),
    )


def channel(value):
    version(value)
    return (
        "nightly"
        if "_ab" in value
        else "prerelease"
        if "beta" in value or "rc" in value
        else "stable"
    )


def valid_path(value):
    check(isinstance(value, str), "invalid artifact path")
    parts = value.split("/")
    check(
        len(parts) in (3, 4) and parts[0] == "docs" and parts[-1] in ARTIFACTS,
        "artifact path outside allowlist",
    )
    check(
        parts[1] == "nightly" or channel(parts[1]) != "nightly",
        "invalid versioned directory",
    )
    check(len(parts) == 3 or parts[2] == "extra", "invalid package directory")
    check(
        parts[-1] != "nightly.json" or value == "docs/nightly/nightly.json",
        "invalid provenance path",
    )
    return value


def encode(obj):
    return (
        json.dumps(obj, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
        + "\n"
    ).encode()


def digest(data):
    return hashlib.sha256(data).hexdigest()


def semantic(obj):
    if isinstance(obj, dict):
        return {k: semantic(v) for k, v in obj.items() if k not in VOLATILE}
    if isinstance(obj, list):
        return [semantic(v) for v in obj]
    return obj


def parse(data):
    def pairs(items):
        result = {}
        for k, v in items:
            check(k not in result, "duplicate JSON property")
            result[k] = v
        return result

    try:
        return json.loads(
            data,
            object_pairs_hook=pairs,
            parse_constant=lambda _: (_ for _ in ()).throw(
                DiscoveryError("nonfinite JSON number")
            ),
        )
    except (ValueError, UnicodeError, RecursionError) as e:
        raise DiscoveryError("invalid JSON artifact") from e


def timestamp(value):
    try:
        t = datetime.fromisoformat(value.replace("Z", "+00:00"))
        check(t.tzinfo is not None, "publication time lacks timezone")
        return t
    except (ValueError, AttributeError, TypeError) as e:
        raise DiscoveryError("invalid publication timestamp") from e


class HTTPSource:
    def __init__(self, attempts=3, timeout=20, deadline=300):
        self.attempts, self.timeout = attempts, timeout
        self.deadline = time.monotonic() + deadline
        self.opener = urllib.request.build_opener(NoRedirect())

    def fetch(self, url):
        for attempt in range(self.attempts):
            remaining = self.deadline - time.monotonic()
            check(remaining > 0, "discovery deadline exceeded")
            try:
                request = urllib.request.Request(
                    url,
                    headers={
                        "User-Agent": "routeros-provider-discovery/1",
                        "Accept": "application/json",
                    },
                )
                with self.opener.open(
                    request, timeout=min(self.timeout, remaining)
                ) as r:
                    length = r.headers.get("Content-Length")
                    check(
                        length is None or (length.isdigit() and int(length) <= LIMIT),
                        "invalid/excessive content length",
                    )
                    chunks, size = [], 0
                    while True:
                        check(
                            time.monotonic() < self.deadline,
                            "discovery deadline exceeded",
                        )
                        chunk = r.read1(min(65536, LIMIT + 1 - size))
                        if not chunk:
                            break
                        chunks.append(chunk)
                        size += len(chunk)
                        check(size <= LIMIT, "artifact exceeds size limit")
                    if length is not None and size != int(length):
                        raise OSError("truncated upstream body")
                    return b"".join(chunks)
            except urllib.error.HTTPError as e:
                if e.code == 404:
                    raise Missing() from None
                if e.code not in (408, 429, 500, 502, 503, 504):
                    raise DiscoveryError(f"upstream HTTP {e.code}") from None
            except (OSError, urllib.error.URLError, http.client.HTTPException):
                pass
            if attempt + 1 < self.attempts:
                time.sleep(min(2**attempt, max(0, self.deadline - time.monotonic())))
        raise DiscoveryError("upstream fetch failed after bounded retries")

    def resolve(self, requested=None):
        if requested is not None:
            check(SHA.fullmatch(requested), "SHA must be full lowercase hexadecimal")
            return requested
        try:
            head = parse(
                self.fetch("https://api.github.com/repos/tikoci/restraml/commits/main")
            )
            check(isinstance(head, dict), "invalid upstream HEAD response")
            value = head.get("sha")
        except Missing:
            raise DiscoveryError("upstream HEAD unavailable") from None
        check(isinstance(value, str) and SHA.fullmatch(value), "invalid upstream HEAD")
        return value

    def get(self, sha, path):
        check(SHA.fullmatch(sha), "invalid upstream SHA")
        check(path == "docs/docs-index.json" or valid_path(path), "invalid path")
        return self.fetch(
            f"https://raw.githubusercontent.com/{REPOSITORY}/{sha}/{path}"
        )


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


class GitSource:
    """Read committed blobs, never mutable files or upstream executable scripts."""

    def __init__(self, directory):
        self.directory = directory

    def git(self, *args):
        return subprocess.run(
            ["git", "-C", self.directory, *args], capture_output=True, timeout=30
        )

    def resolve(self, requested=None):
        if requested is not None:
            check(SHA.fullmatch(requested), "invalid SHA")
        result = self.git("rev-parse", "--verify", (requested or "HEAD") + "^{commit}")
        check(result.returncode == 0, "cannot resolve committed snapshot")
        sha = result.stdout.decode().strip()
        check(SHA.fullmatch(sha), "invalid commit SHA")
        return sha

    def get(self, sha, path):
        check(path == "docs/docs-index.json" or valid_path(path), "invalid path")
        result = self.git("cat-file", "-s", sha + ":" + path)
        if result.returncode:
            raise Missing()
        check(
            result.stdout.strip().isdigit() and int(result.stdout) <= LIMIT,
            "artifact exceeds size limit",
        )
        result = self.git("show", sha + ":" + path)
        check(
            result.returncode == 0 and len(result.stdout) <= LIMIT,
            "cannot read snapshot blob",
        )
        return result.stdout


def inventory(index):
    check(
        isinstance(index, dict)
        and index.get("format") == "restraml-docs-index@1"
        and index.get("rootPath") == "docs",
        "unsupported index format/root",
    )
    timestamp(index.get("generatedAt"))
    stable, latest = (
        version(index.get("latestStableVersion")),
        version(index.get("latestVersion")),
    )
    check(
        channel(stable) == "stable" and channel(latest) != "nightly",
        "invalid latest pointers",
    )
    check(
        isinstance(index.get("versions"), list) and len(index["versions"]) <= 10000,
        "invalid index versions",
    )
    found = {}
    for entry in index["versions"]:
        check(isinstance(entry, dict), "invalid index entry")
        name = entry.get("name")
        check(
            name == "nightly" or channel(name) != "nightly", "invalid indexed version"
        )
        check(
            name not in found
            and entry.get("path") == f"docs/{name}"
            and entry.get("type") == "dir",
            "duplicate/invalid version directory",
        )
        paths = set()

        def visit(node, parent, depth=0, paths=paths):
            check(
                depth <= 1
                and isinstance(node.get("files"), list)
                and isinstance(node.get("dirs"), list),
                "invalid index tree",
            )
            for item in node["files"]:
                check(
                    isinstance(item, dict) and item.get("type") == "file",
                    "invalid file entry",
                )
                leaf = item.get("name")
                check(
                    isinstance(leaf, str)
                    and re.fullmatch(r"[A-Za-z0-9_.-]+", leaf)
                    and leaf not in (".", ".."),
                    "unsafe file name",
                )
                path = parent + "/" + leaf
                check(
                    item.get("path") == path and path not in paths,
                    "invalid/duplicate indexed path",
                )
                paths.add(path)
            for child in node["dirs"]:
                check(
                    isinstance(child, dict)
                    and child.get("name") == "extra"
                    and child.get("path") == parent + "/extra"
                    and child.get("type") == "dir",
                    "unsupported indexed directory",
                )
                visit(child, parent + "/extra", depth + 1)

        visit(entry, entry["path"])
        found[name] = paths
    check(stable in found and latest in found, "latest pointer absent from index")
    return stable, latest, found


def validate_spec(spec, selected_version, path):
    check(
        isinstance(spec, dict)
        and spec.get("openapi") in ("3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1.0")
        and isinstance(spec.get("info"), dict)
        and spec["info"].get("version") == selected_version
        and isinstance(spec.get("paths"), dict)
        and spec["paths"],
        "malformed selected schema or info.version mismatch: " + path,
    )
    operations = 0
    for route, item in spec["paths"].items():
        if route.startswith("x-"):
            continue
        check(
            route.startswith("/") and isinstance(item, dict),
            "malformed OpenAPI path item",
        )
        for method in (
            "get",
            "put",
            "post",
            "delete",
            "options",
            "head",
            "patch",
            "trace",
        ):
            if method in item:
                operation = item[method]
                check(
                    isinstance(operation, dict)
                    and isinstance(operation.get("responses"), dict)
                    and operation["responses"],
                    "malformed OpenAPI operation responses",
                )
                operations += 1
    check(operations > 0, "selected schema has no operations")


def fingerprints(root):
    provenance = parse((root / "schemas/provenance.json").read_bytes())
    files = [
        root / "go.mod",
        root / "go.sum",
        root / "tools/go.mod",
        root / "tools/go.sum",
        root / "schemas/ip-address-policy.json",
        root / "internal/catalog/collections.json",
        root / "schemas/generator-config.yml",
        root / "tools/schema/discover/discover.py",
    ]
    for name in (
        "schemas/maintenance-policy.json",
        "schemas/wire-descriptors.json",
        "internal/catalog/singletons.json",
        "internal/catalog/extended-collections.json",
        "tools/schema/bindings.py",
        "internal/catalog/additional-settings.json",
        "tools/schema/additional_settings_bindings.py",
    ):
        if (root / name).is_file():
            files.append(root / name)
    for directory in (
        "internal/client",
        "internal/provider",
        "internal/catalog",
        "tools/schema/normalize",
        "tools/schema/validate",
    ):
        files.extend(
            p
            for pattern in ("*.go", "*.py")
            for p in (root / directory).rglob(pattern)
            if not p.name.endswith("_test.go")
            and not p.name.startswith("test_")
            and "fixtures" not in p.parts
        )
    files.append(root / "tools/schema/generate.sh")
    # Maintenance reuse must invalidate when offline verification/orchestration changes,
    # not merely when schema producers change. Never hash ephemeral caches/credentials.
    verification_files = [root / "GNUmakefile"]
    for directory in ("internal", "tools", ".github/workflows"):
        for p in (root / directory).rglob("*"):
            if (
                p.is_file()
                and not any(
                    part in ("bin", "__pycache__", ".local")
                    for part in p.relative_to(root).parts
                )
                and (
                    p.name.endswith("_test.go")
                    or p.suffix in (".py", ".yml", ".json")
                    or (directory == "tools" and p.suffix == ".go")
                )
            ):
                verification_files.append(p)
    package_policies = [
        root / "internal/catalog" / name
        for name in ("extended-collections.json", "additional-settings.json")
    ]
    requires_extra = any(
        p.is_file()
        and any(row.get("required_package") for row in parse(p.read_bytes()))
        for p in package_policies
    )
    return {
        "requires_extra_companion": bool(requires_extra),
        "provenance": provenance,
        "sources": {
            str(p.relative_to(root)): digest(p.read_bytes()) for p in sorted(files)
        },
        "verification": {
            str(p.relative_to(root)): digest(p.read_bytes())
            for p in sorted(verification_files)
        },
    }


def discover(
    source,
    inputs,
    state,
    *,
    requested=None,
    backfill=0,
    extras=False,
    nightly=False,
    replay=False,
    force=False,
    now=None,
    max_age_days=7,
):
    check(0 <= backfill <= 8, "backfill outside 0..8")
    now = now or datetime.now(UTC)
    sha = source.resolve(requested)  # The only HEAD resolution in this run.
    cache = {}

    def fetch(path):
        if path not in cache:
            cache[path] = source.get(sha, path)
        return cache[path]

    try:
        index_raw = fetch("docs/docs-index.json")
    except Missing:
        raise DiscoveryError("snapshot index missing") from None
    index = parse(index_raw)
    stable, latest, indexed = inventory(index)
    names = {stable, latest}
    others = sorted(
        (n for n in indexed if n != "nightly" and n not in names),
        key=order,
        reverse=True,
    )
    names.update(others[:backfill])
    if nightly:
        names.add("nightly")
    candidates, deferred = [], []
    for name in sorted(
        names, key=lambda n: (n == "nightly", () if n == "nightly" else order(n))
    ):
        publication = None
        selected_version = name
        if name == "nightly":
            try:
                publication = parse(fetch("docs/nightly/nightly.json"))
            except Missing:
                deferred.append(
                    {"path": "docs/nightly/nightly.json", "reason": "not-yet-published"}
                )
                continue
            check(isinstance(publication, dict), "invalid nightly provenance")
            selected_version = version(publication.get("nightlyVersion"))
            check(channel(selected_version) == "nightly", "invalid nightly version")
            index_nightly = index.get("nightly")
            check(
                isinstance(index_nightly, dict)
                and index_nightly.get("nightlyVersion") == selected_version,
                "nightly index/provenance mismatch",
            )
            built = timestamp(publication.get("builtAt"))
            check(
                timestamp(index_nightly.get("builtAt")) == built,
                "nightly index publication time mismatch",
            )
            check(
                replay
                or (
                    now - timedelta(days=max_age_days)
                    <= built
                    <= now + timedelta(minutes=5)
                ),
                "stale/future nightly publication (explicit replay required)",
            )
            provenance = publication.get("provenance")
            check(
                isinstance(provenance, dict)
                and isinstance(provenance.get("x86"), dict)
                and "base" in provenance["x86"],
                "missing canonical nightly provenance",
            )
            for arch, phases in provenance.items():
                check(
                    arch in ("x86", "arm64") and isinstance(phases, dict) and phases,
                    "invalid provenance architecture",
                )
                for phase, p in phases.items():
                    check(
                        phase in ("base", "extra")
                        and isinstance(p, dict)
                        and p.get("arch") == arch
                        and p.get("phase") == phase,
                        "invalid nightly phase provenance",
                    )
                    check(
                        p.get("nightlyVersion") == selected_version
                        and p.get("postVersion") == selected_version,
                        "mixed nightly provenance versions",
                    )
                    phase_time = timestamp(p.get("builtAt"))
                    check(phase_time <= built, "phase published after aggregate")
                    check(
                        replay or now - timedelta(days=max_age_days) <= phase_time,
                        "stale nightly phase (explicit replay required)",
                    )
        for flavor in ("base", "extra") if extras else ("base",):
            directory = f"docs/{name}" + ("/extra" if flavor == "extra" else "")
            path = valid_path(directory + "/openapi.json")
            try:
                raw = fetch(path)  # Actual existence, not index readiness guesses.
            except Missing:
                deferred.append({"path": path, "reason": "not-yet-published"})
                continue
            spec = parse(raw)
            validate_spec(spec, selected_version, path)
            metadata = {}
            if flavor == "base" and inputs.get("requires_extra_companion"):
                companion = valid_path(f"docs/{name}/extra/openapi.json")
                try:
                    companion_data = fetch(companion)
                except Missing:
                    deferred.append(
                        {
                            "path": companion,
                            "reason": "reviewed-package-companion-not-yet-published",
                        }
                    )
                    continue
                companion_spec = parse(companion_data)
                validate_spec(companion_spec, selected_version, companion)
                metadata[companion] = {
                    "sha256": digest(companion_data),
                    "semantic_sha256": digest(encode(semantic(companion_spec))),
                }
            for leaf in (
                "inspect.json",
                "deep-inspect.json",
                "deep-inspect.x86.json",
                "deep-inspect.arm64.json",
            ):
                artifact = directory + "/" + leaf
                if artifact not in indexed.get(name, set()):
                    continue
                try:
                    data = fetch(valid_path(artifact))
                except Missing:
                    raise DiscoveryError(
                        "indexed metadata missing: " + artifact
                    ) from None
                obj = parse(data)
                check(isinstance(obj, dict), "invalid inspect metadata")
                if leaf.startswith("deep-inspect"):
                    meta = obj.get("_meta")
                    check(
                        isinstance(meta, dict)
                        and meta.get("version") == selected_version,
                        "inspect version mismatch",
                    )
                    expected_arch = "arm64" if "arm64" in leaf else "x86"
                    check(
                        meta.get("architecture", expected_arch) == expected_arch,
                        "inspect architecture mismatch",
                    )
                metadata[artifact] = {
                    "sha256": digest(data),
                    "semantic_sha256": digest(encode(semantic(obj))),
                }
            if publication:
                check(
                    flavor in publication["provenance"]["x86"],
                    "missing selected nightly phase",
                )
                metadata["docs/nightly/nightly.json"] = {
                    "sha256": digest(fetch("docs/nightly/nightly.json")),
                    "semantic_sha256": digest(encode(semantic(publication))),
                }
            lane = channel(selected_version)
            identity = {
                "repository": REPOSITORY,
                "channel": lane,
                "version": selected_version,
                "flavor": flavor,
                "architecture": "x86",
                "schema_path": path,
                "schema_sha256": digest(raw),
                "semantic_metadata": {
                    p: v["semantic_sha256"] for p, v in sorted(metadata.items())
                },
                "inputs": inputs,
            }
            key = digest(encode(identity))
            selector = (
                "nightly"
                if name == "nightly"
                else "stable"
                if name == stable
                else "latest"
                if name == latest
                else "backfill"
            )
            lane_key = selector + "/" + flavor + "/x86"
            previous = (
                state.get("lanes", {}).get(lane_key) if selector != "backfill" else None
            )
            if previous and not replay:
                check(
                    order(selected_version) >= order(previous["version"]),
                    "regressing RouterOS version (explicit replay required)",
                )
                if publication and previous.get("built_at"):
                    check(
                        timestamp(publication["builtAt"])
                        >= timestamp(previous["built_at"]),
                        "regressing nightly publication",
                    )
            stages = state.get("candidates", {}).get(key, {}).get("stages", {})
            candidates.append(
                {
                    **identity,
                    "key": key,
                    "upstream_sha": sha,
                    "metadata": metadata,
                    "publication": publication,
                    "publication_metadata": {
                        "index_generated_at": index["generatedAt"],
                        "nightly_built_at": publication["builtAt"]
                        if publication
                        else None,
                    },
                    "lane": lane_key,
                    "tracked": selector != "backfill",
                    "stages": stages,
                    "generation_needed": force or "generated" not in stages,
                }
            )
    # Keep raw hashes in the audit manifest but not volatile index timestamps in keys.
    report = {
        "format": FORMAT,
        "repository": REPOSITORY,
        "upstream_sha": sha,
        "index_sha256": digest(index_raw),
        "index_generated_at": index["generatedAt"],
        "replay": replay,
        "force": force,
        "candidates": sorted(
            candidates, key=lambda c: (c["channel"], c["version"], c["flavor"])
        ),
        "deferred": sorted(deferred, key=lambda d: d["path"]),
    }
    report["outcome"] = (
        "changed"
        if any(c["generation_needed"] for c in candidates)
        else "not-yet-published"
        if deferred
        else "unchanged"
    )
    return report, cache


def record_discovery(state, report):
    updated = parse(encode(state))
    updated.setdefault("format", FORMAT)
    for c in report["candidates"]:
        entry = updated.setdefault("candidates", {}).setdefault(
            c["key"], {"stages": {}}
        )
        entry["stages"]["discovered"] = {"upstream_sha": report["upstream_sha"]}
        entry["identity"] = {
            k: c[k]
            for k in ("version", "channel", "flavor", "architecture", "schema_sha256")
        }
        if not report["replay"] and c["tracked"]:
            updated.setdefault("lanes", {})[c["lane"]] = {
                "version": c["version"],
                "built_at": c["publication"]["builtAt"] if c["publication"] else None,
            }
    return updated


def advance(state, key, stage, evidence):
    check(stage in STAGES[1:], "invalid checkpoint stage")
    entry = state.get("candidates", {}).get(key)
    check(entry is not None, "candidate was not discovered")
    proof = parse(evidence)
    check(
        isinstance(proof, dict)
        and proof.get("candidate_key") == key
        and proof.get("stage") == stage
        and proof.get("success") is True,
        "checkpoint requires matching successful stage evidence",
    )
    stages = entry["stages"]
    check(STAGES[STAGES.index(stage) - 1] in stages, "previous stage has not succeeded")
    receipt = {"evidence_sha256": digest(evidence)}
    check(
        stage not in stages or stages[stage] == receipt,
        "cannot overwrite successful stage receipt",
    )
    stages[stage] = receipt


def atomic(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + ".tmp")
    with temporary.open("wb") as f:
        f.write(data)
        f.flush()
        os.fsync(f.fileno())
    os.replace(temporary, path)


@contextmanager
def locked(path):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.with_name(path.name + ".lock").open("a") as f:
        fcntl.flock(f, fcntl.LOCK_EX)
        yield


def load_state(path):
    state: dict[str, Any] = (
        parse(path.read_bytes())
        if path.exists()
        else {"format": FORMAT, "candidates": {}, "lanes": {}}
    )
    check(
        isinstance(state, dict)
        and state.get("format") == FORMAT
        and isinstance(state.get("candidates"), dict)
        and isinstance(state.get("lanes"), dict),
        "invalid checkpoint state",
    )
    for key, entry in state["candidates"].items():
        check(
            re.fullmatch(r"[0-9a-f]{64}", key)
            and isinstance(entry, dict)
            and isinstance(entry.get("stages"), dict),
            "invalid candidate checkpoint",
        )
        stages = entry["stages"]
        check(set(stages) <= set(STAGES), "unknown checkpoint stage")
        for stage, receipt in stages.items():
            check(isinstance(receipt, dict), "invalid stage receipt")
            if stage == "discovered":
                check(
                    isinstance(receipt.get("upstream_sha"), str)
                    and SHA.fullmatch(receipt["upstream_sha"]),
                    "invalid discovery receipt",
                )
            else:
                check(
                    STAGES[STAGES.index(stage) - 1] in stages
                    and isinstance(receipt.get("evidence_sha256"), str)
                    and re.fullmatch(r"[0-9a-f]{64}", receipt["evidence_sha256"]),
                    "invalid stage receipt/order",
                )
    for lane, frontier in state["lanes"].items():
        check(
            re.fullmatch(r"(stable|latest|nightly)/(base|extra)/x86", lane)
            and isinstance(frontier, dict),
            "invalid lane checkpoint",
        )
        version(frontier.get("version"))
        if frontier.get("built_at") is not None:
            timestamp(frontier["built_at"])
    return state


def main():
    root = Path(__file__).resolve().parents[3]
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("run")
    p.add_argument("--sha")
    p.add_argument(
        "--repository-dir", help="read-only local Git object store instead of network"
    )
    p.add_argument("--backfill", type=int, default=0)
    p.add_argument("--extra", action="store_true")
    p.add_argument("--nightly", action="store_true")
    p.add_argument("--replay", action="store_true")
    p.add_argument("--force", action="store_true")
    p.add_argument("--output", type=Path, default=root / ".local/discovery")
    p.add_argument(
        "--state", type=Path, default=root / ".local/discovery-checkpoints.json"
    )
    p = sub.add_parser("checkpoint")
    p.add_argument("--key", required=True)
    p.add_argument("--stage", choices=STAGES[1:], required=True)
    p.add_argument("--evidence", type=Path, required=True)
    p.add_argument(
        "--state", type=Path, default=root / ".local/discovery-checkpoints.json"
    )
    args = parser.parse_args()
    try:
        with locked(args.state):
            state = load_state(args.state)
            if args.command == "checkpoint":
                advance(state, args.key, args.stage, args.evidence.read_bytes())
                atomic(args.state, encode(state))
                return
            source = (
                GitSource(args.repository_dir) if args.repository_dir else HTTPSource()
            )
            report, blobs = discover(
                source,
                fingerprints(root),
                state,
                requested=args.sha,
                backfill=args.backfill,
                extras=args.extra,
                nightly=args.nightly,
                replay=args.replay,
                force=args.force,
            )
            # Commit outputs only after every selected artifact has been validated.
            for path, data in sorted(blobs.items()):
                target = args.output / report["upstream_sha"] / path
                check(
                    not target.exists() or target.read_bytes() == data,
                    "immutable snapshot output already has different bytes",
                )
            for path, data in sorted(blobs.items()):
                atomic(args.output / report["upstream_sha"] / path, data)
            atomic(args.output / "manifest.json", encode(report))
            atomic(args.state, encode(record_discovery(state, report)))
            print(
                json.dumps(
                    {
                        "outcome": report["outcome"],
                        "upstream_sha": report["upstream_sha"],
                        "candidates": len(report["candidates"]),
                        "deferred": len(report["deferred"]),
                    },
                    sort_keys=True,
                )
            )
    except (DiscoveryError, OSError, subprocess.TimeoutExpired) as e:
        print("discovery-failure: " + str(e), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
