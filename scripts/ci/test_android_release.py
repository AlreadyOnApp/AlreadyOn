"""Release policy tests use synthetic APK bytes and mocked GitHub operations."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("android_release", Path(__file__).with_name("android-release.py"))
release = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release)


class AndroidReleaseTests(unittest.TestCase):
    def fixture(self, directory):
        (directory / release.ASSETS[0]).write_bytes(b"synthetic signed APK")
        (directory / release.ASSETS[3]).write_bytes(b"synthetic signed TV APK")
        with patch.object(release, "publish"), patch.object(release, "releases", return_value=[]), \
                patch.object(release, "changelog", return_value="## Features\n\n- Synthetic feature."):
            release.candidate("owner/repo", "2026-10-09-rc2", "a" * 40, 200000000, directory)

    def test_download_includes_both_apks_and_provenance(self):
        with patch.object(release, "run") as command:
            release.download("owner/repo", "2026-10-09-rc1", Path("assets"))
            arguments = command.call_args.args
            for asset in release.ASSETS:
                self.assertIn(asset, arguments)

    def test_tv_checksum_detects_changed_apk(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            self.fixture(directory)
            (directory / release.ASSETS[3]).write_bytes(b"different TV APK")
            with self.assertRaisesRegex(ValueError, "TV APK checksum"):
                release.verify_assets(directory)

    def test_candidate_numbers_include_existing_tags_and_drafts(self):
        self.assertEqual(release.next_tag("2026-10-09", []), "2026-10-09-rc1")
        self.assertEqual(release.next_tag("2026-10-09", ["2026-10-09-rc1", "2026-10-09-rc3",
                                                       "2026-10-08-rc9", "2026-10-09-stable"]),
                         "2026-10-09-rc4")

    def test_checksums_detect_changed_apk(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            self.fixture(directory)
            release.verify_assets(directory, "a" * 40)
            (directory / release.ASSETS[0]).write_bytes(b"different APK")
            with self.assertRaisesRegex(ValueError, "checksum"):
                release.verify_assets(directory, "a" * 40)

    def test_checksums_detect_changed_manifest(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            self.fixture(directory)
            path = directory / release.ASSETS[2]
            data = json.loads(path.read_text())
            data["version_code"] += 1
            path.write_text(json.dumps(data))
            with self.assertRaisesRegex(ValueError, "checksum"):
                release.verify_assets(directory, "a" * 40)

    def test_provenance_must_match_tag_commit(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            self.fixture(directory)
            with self.assertRaisesRegex(ValueError, "revision"):
                release.verify_assets(directory, "b" * 40)

    def test_stable_reuses_exact_candidate_bytes_and_commit(self):
        def download(repo, tag, directory):
            self.fixture(directory)
        def publish(repo, tag, sha, directory, notes, prerelease):
            self.assertEqual(tag, "2026-10-09-stable")
            self.assertEqual(sha, "a" * 40)
            self.assertFalse(prerelease)
            self.assertIn("- Tested feature.", notes)
            self.assertEqual((directory / release.ASSETS[0]).read_bytes(), b"synthetic signed APK")
            self.assertEqual(release.verify_assets(directory)["version_name"], "2026-10-09-rc2")
        with patch.object(release, "api", return_value={"draft": False, "prerelease": True, "body": "<!-- end-user-changelog:start -->\n## Features\n\n- Tested feature.\n<!-- end-user-changelog:end -->"}), \
                patch.object(release, "run", return_value="a" * 40), \
                patch.object(release.subprocess, "run") as git, \
                patch.object(release, "download", side_effect=download), \
                patch.object(release, "publish", side_effect=publish) as published:
            release.promote("owner/repo", "2026-10-09-rc2")
            published.assert_called_once()
            git.assert_called_once_with(["git", "merge-base", "--is-ancestor", "a" * 40, "origin/main"], check=True)

    def test_stable_rejects_drafts_and_non_candidates(self):
        for data in ({"draft": True, "prerelease": True}, {"draft": False, "prerelease": False}):
            with patch.object(release, "api", return_value=data), patch.object(release, "publish") as published:
                with self.assertRaises(ValueError):
                    release.promote("owner/repo", "2026-10-09-rc1")
                published.assert_not_called()
        with self.assertRaises(ValueError):
            release.promote("owner/repo", "main")

    def test_existing_published_release_must_match_bytes(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            self.fixture(directory)
            def download(repo, tag, remote):
                self.fixture(remote)
                (remote / release.ASSETS[0]).write_bytes(b"previous APK")
            with patch.object(release, "releases", return_value=[{"tag_name": "2026-10-09-rc2", "draft": False}]), \
                    patch.object(release, "api", return_value={"sha": "a" * 40}), \
                    patch.object(release, "download", side_effect=download), \
                    patch.object(release, "run") as command:
                with self.assertRaisesRegex(ValueError, "overwrite"):
                    release.publish("owner/repo", "2026-10-09-rc2", "a" * 40, directory, "notes", True)
                command.assert_not_called()

    def test_completed_rerun_verifies_and_reuses_candidate(self):
        sha = "a" * 40
        entry = {"tag_name": "2026-10-09-rc2", "draft": False,
                 "body": f"Source commit: `{sha}`"}
        def download(repo, tag, directory):
            self.fixture(directory)
        with patch.object(release, "releases", return_value=[entry]), \
                patch.object(release, "api", return_value={"sha": sha}), \
                patch.object(release, "download", side_effect=download), \
                patch("builtins.print") as output:
            release.plan("owner/repo", sha)
            output.assert_any_call("existing=true")
            output.assert_any_call("tag=2026-10-09-rc2")

    def test_failed_draft_reuses_original_android_version_code(self):
        sha = "a" * 40
        entry = {"tag_name": "2026-10-09-rc2", "draft": True, "target_commitish": sha,
                 "body": f"Source commit: `{sha}`\nAndroid versionCode: `200000000`"}
        with patch.object(release, "releases", return_value=[entry]), \
                patch.object(release, "api", return_value={"sha": sha}), \
                patch("builtins.print") as output:
            release.plan("owner/repo", sha)
            output.assert_any_call("existing=false")
            output.assert_any_call("version_code=200000000")

    def test_existing_tag_must_match_commit(self):
        with patch.object(release, "releases", return_value=[{"tag_name": "2026-10-09-stable", "draft": True}]), \
                patch.object(release, "api", return_value={"sha": "b" * 40}):
            with self.assertRaisesRegex(ValueError, "different source"):
                release.publish("owner/repo", "2026-10-09-stable", "a" * 40, Path("unused"), "notes", False)


if __name__ == "__main__":
    unittest.main()
