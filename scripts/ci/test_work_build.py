"""Exercise isolated issue-branch creation against local Git repositories."""
import pathlib
import subprocess
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).resolve().parents[1] / 'work.py'


class WorkBranchTest(unittest.TestCase):
    def test_start_assistant_branches_from_current_dev(self):
        with tempfile.TemporaryDirectory() as root:
            root = pathlib.Path(root)
            remote, repo = root / 'remote.git', root / 'work'
            subprocess.run(['git', 'init', '--bare', str(remote)], check=True, capture_output=True)
            repo.mkdir()

            def git(*args):
                return subprocess.check_output(['git', *args], cwd=repo, stderr=subprocess.DEVNULL, text=True).strip()

            git('init', '-b', 'dev')
            git('config', 'user.name', 'Fixture')
            git('config', 'user.email', 'fixture@example.invalid')
            git('commit', '--allow-empty', '-m', 'baseline')
            git('remote', 'add', 'origin', str(remote))
            git('push', '-u', 'origin', 'dev')

            def start(name):
                return subprocess.check_output(['python3', str(SCRIPT), 'start', name], cwd=repo, text=True)

            start('build')
            self.assertEqual(git('branch', '--show-current'), 'build')
            git('commit', '--allow-empty', '-m', 'pending fix')
            head = git('rev-parse', 'HEAD')
            git('switch', 'dev')
            start('build')
            self.assertEqual(git('rev-parse', 'HEAD'), head)

    def test_assistant_description_must_be_two_or_three_words(self):
        with tempfile.TemporaryDirectory() as root:
            root = pathlib.Path(root)
            remote, repo = root / 'remote.git', root / 'work'
            subprocess.run(['git', 'init', '--bare', str(remote)], check=True, capture_output=True)
            repo.mkdir()
            subprocess.run(['git', 'init', '-b', 'dev'], cwd=repo, check=True, capture_output=True)
            subprocess.run(['git', 'remote', 'add', 'origin', str(remote)], cwd=repo, check=True)
            result = subprocess.run(
                ['python3', str(SCRIPT), 'start', 'codex/too-many-description-words'],
                cwd=repo, text=True, capture_output=True,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('invalid choice', result.stderr)


if __name__ == '__main__':
    unittest.main()
