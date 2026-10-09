#!/usr/bin/env python3
"""Publish signed Android candidates and copy tested assets for stable promotion."""
import argparse
from datetime import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time
from zoneinfo import ZoneInfo

RC = re.compile(r"(\d{4}-\d{2}-\d{2})-rc([1-9]\d*)\Z")
ASSETS = ("alreadyon-android.apk", "SHA256SUMS", "android-release.json", "alreadyon-tv.apk")


def run(*args):
    return subprocess.check_output(args, text=True).strip()


def api(endpoint):
    return json.loads(run("gh", "api", endpoint))


def releases(repo):
    pages = json.loads(run("gh", "api", "--paginate", "--slurp",
                           f"repos/{repo}/releases?per_page=100"))
    return [release for page in pages for release in page]


def next_tag(day, tags):
    numbers = [int(match[2]) for tag in tags
               if (match := RC.fullmatch(tag)) and match[1] == day]
    return f"{day}-rc{max(numbers, default=0) + 1}"


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify_assets(directory, source=None):
    manifest = json.loads((directory / "android-release.json").read_text())
    if manifest.get("package") != "io.github.lynchbros.alreadyon":
        raise ValueError("Unexpected Android application identity")
    if source is not None and manifest.get("source_sha") != source:
        raise ValueError("Candidate source revision does not match its tag")
    if not RC.fullmatch(manifest.get("version_name", "")):
        raise ValueError("Invalid candidate version")
    expected = digest(directory / ASSETS[0])
    if manifest.get("apk_sha256") != expected:
        raise ValueError("APK checksum does not match release provenance")
    if manifest.get("tv_apk_sha256") != digest(directory / ASSETS[3]):
        raise ValueError("TV APK checksum does not match release provenance")
    sums = (directory / "SHA256SUMS").read_text()
    required = "".join(f"{digest(directory / name)}  {name}\n"
                       for name in (ASSETS[0], ASSETS[2], ASSETS[3]))
    if sums != required:
        raise ValueError("Release checksum file does not match the assets")
    return manifest


def download(repo, tag, directory):
    run("gh", "release", "download", tag, "--repo", repo, "--dir", str(directory),
        *(argument for name in ASSETS for argument in ("--pattern", name)))


def publish(repo, tag, sha, directory, notes, prerelease):
    existing = [r for r in releases(repo) if r["tag_name"] == tag]
    if existing:
        # GitHub may defer creating a draft's tag until publication.
        tagged_sha = (existing[0].get("target_commitish") if existing[0]["draft"]
                      else api(f"repos/{repo}/commits/{tag}")["sha"])
        if tagged_sha != sha:
            raise ValueError("Existing release tag points to a different source commit")
    if existing and not existing[0]["draft"]:
        # A rerun may verify a published release, but must never replace its assets.
        with tempfile.TemporaryDirectory() as temp:
            remote = Path(temp)
            download(repo, tag, remote)
            if any(digest(remote / name) != digest(directory / name) for name in ASSETS):
                raise ValueError("Published release differs; refusing to overwrite it")
        return
    with tempfile.TemporaryDirectory() as temp:
        body = Path(temp) / "notes.md"
        body.write_text(notes)
        if existing:
            run("gh", "release", "edit", tag, "--repo", repo, "--notes-file", str(body))
        else:
            run("gh", "release", "create", tag, "--repo", repo, "--target", sha,
                "--title", f"AlreadyOn {tag}", "--notes-file", str(body), "--draft")
        run("gh", "release", "upload", tag, "--repo", repo,
            *(str(directory / name) for name in ASSETS), "--clobber")
        # Verify uploaded bytes before exposing the draft as a public release.
        remote = Path(temp) / "verify"
        remote.mkdir()
        download(repo, tag, remote)
        if any(digest(remote / name) != digest(directory / name) for name in ASSETS):
            raise ValueError("Uploaded release assets did not match local bytes")
        run("gh", "release", "edit", tag, "--repo", repo, "--draft=false",
            f"--prerelease={'true' if prerelease else 'false'}",
            f"--latest={'false' if prerelease else 'true'}")


def plan(repo, sha):
    entries = releases(repo)
    marker = f"Source commit: `{sha}`"
    existing = next((r for r in entries if RC.fullmatch(r["tag_name"])
                     and marker in (r.get("body") or "")), None)
    if existing:
        target = (existing.get("target_commitish") if existing["draft"]
                  else api(f"repos/{repo}/commits/{existing['tag_name']}")["sha"])
        if target != sha:
            raise ValueError("Existing candidate tag does not match its recorded source")
    if existing and not existing["draft"]:
        with tempfile.TemporaryDirectory() as temp:
            download(repo, existing["tag_name"], Path(temp))
            verify_assets(Path(temp), sha)
        print("existing=true")
        print(f"tag={existing['tag_name']}")
        return
    if existing:
        code = re.search(r"Android versionCode: `(\d+)`", existing.get("body") or "")
        if code is None:
            raise ValueError("Incomplete draft has no Android version code")
        tag, version_code = existing["tag_name"], int(code[1])
    else:
        day = datetime.now(ZoneInfo("America/New_York")).date().isoformat()
        tags = run("git", "tag", "--list").splitlines()
        tag = next_tag(day, tags + [r["tag_name"] for r in entries])
        # Seconds since 2020 fit Android's signed-int limit and increase between builds.
        version_code = int(time.time()) - 1577836800
    print("existing=false")
    print(f"tag={tag}")
    print(f"version_code={version_code}")


def candidate(repo, tag, sha, code, directory):
    if not RC.fullmatch(tag) or not 0 < code <= 2100000000:
        raise ValueError("Invalid Android release version")
    manifest = {"source_sha": sha, "version_name": tag, "version_code": code,
                "package": "io.github.lynchbros.alreadyon",
                "apk_sha256": digest(directory / ASSETS[0]),
                "tv_apk_sha256": digest(directory / ASSETS[3])}
    (directory / ASSETS[2]).write_text(json.dumps(manifest, indent=2) + "\n")
    (directory / ASSETS[1]).write_text("".join(
        f"{digest(directory / name)}  {name}\n" for name in (ASSETS[0], ASSETS[2], ASSETS[3])))
    verify_assets(directory, sha)
    notes = (f"Android mobile and Google TV release candidate `{tag}`.\n\n"
             f"Source commit: `{sha}`\n\nAndroid versionCode: `{code}`\n\n"
             "Signed release package: `io.github.lynchbros.alreadyon`. "
             "The developer `.debug` app remains a separate installation.\n\n"
             "Test mobile and Google TV builds on devices before stable promotion. "
             "Checksums and source provenance are attached.\n")
    publish(repo, tag, sha, directory, notes, True)


def promote(repo, tag):
    match = RC.fullmatch(tag)
    if match is None:
        raise ValueError("Stable promotion requires a YYYY-MM-DD-rcN candidate")
    release = api(f"repos/{repo}/releases/tags/{tag}")
    if release["draft"] or not release["prerelease"]:
        raise ValueError("Select a published Android release candidate")
    # The checked-out full history includes tags; never trust target_commitish as a SHA.
    sha = run("git", "rev-parse", f"refs/tags/{tag}^{{commit}}")
    subprocess.run(["git", "merge-base", "--is-ancestor", sha, "origin/main"], check=True)
    with tempfile.TemporaryDirectory() as temp:
        directory = Path(temp)
        download(repo, tag, directory)
        manifest = verify_assets(directory, sha)
        if manifest["version_name"] != tag:
            raise ValueError("Candidate manifest version does not match the selected tag")
        stable = f"{match[1]}-stable"
        notes = (f"Stable Android mobile and Google TV release promoted from `{tag}`.\n\n"
                 f"Source commit: `{sha}`\n\n"
                 "Assets are byte-for-byte copies of the tested candidate; nothing was rebuilt "
                 "or re-signed. The APK's embedded version name remains the candidate version.\n")
        publish(repo, stable, sha, directory, notes, False)
        print(f"Published {stable} from {tag}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operation", choices=("plan", "candidate", "promote"))
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY"), required=False)
    parser.add_argument("--sha")
    parser.add_argument("--tag")
    parser.add_argument("--version-code", type=int)
    parser.add_argument("--directory", type=Path)
    args = parser.parse_args()
    if not args.repo:
        parser.error("--repo or GITHUB_REPOSITORY is required")
    if args.operation == "plan":
        if not args.sha:
            parser.error("plan requires --sha")
        plan(args.repo, args.sha)
    elif args.operation == "candidate":
        if not all((args.tag, args.sha, args.version_code, args.directory)):
            parser.error("candidate requires --tag, --sha, --version-code and --directory")
        candidate(args.repo, args.tag, args.sha, args.version_code, args.directory)
    else:
        if not args.tag:
            parser.error("promote requires --tag")
        promote(args.repo, args.tag)


if __name__ == "__main__":
    main()
