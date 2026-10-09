"""Verify local deployment passes through an available Intel/AMD render device."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).parents[2]


class ContainerDeviceTests(unittest.TestCase):
    """Exercise the deploy command with a stub engine, never replacing a container."""

    def test_run_passes_dri_only_when_the_host_has_it(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            media = root / 'media'
            media.mkdir()
            args_file = root / 'run-args'
            engine = root / 'engine'
            engine.write_text('''#!/bin/sh
case "$1" in
  --version) echo 'Docker test engine' ;;
  build|rm) exit 0 ;;
  inspect) echo previous-image ;;
  run) printf '%s\\n' "$@" > "$CONTAINER_TEST_ARGS"; echo test-container-id ;;
  cp) touch "$3" ;;
esac
''')
            engine.chmod(0o755)
            stub_bin = root / 'bin'
            stub_bin.mkdir()
            for name in ('curl', 'systemctl'):
                stub = stub_bin / name
                stub.write_text('#!/bin/sh\nexit 0\n')
                stub.chmod(0o755)
            env = os.environ | {
                'CONTAINER_ENGINE': str(engine),
                'CONTAINER_TEST_ARGS': str(args_file),
                'CONTAINER_IMAGE': 'alreadyon:test',
                'LOCAL_CONTAINER_NAME': 'alreadyon-test',
                'LOCAL_WEB_PORT': '29900',
                'LOCAL_MEDIA_DIR': str(media),
                'PATH': f'{stub_bin}:{os.environ["PATH"]}',
            }

            subprocess.run(['bash', 'scripts/container.sh', 'deploy'], cwd=ROOT,
                           env=env, check=True, capture_output=True, text=True)

            args = args_file.read_text().splitlines()
            has_dri_device = any(arg.startswith('--device=/dev/dri:') for arg in args)
            self.assertEqual(has_dri_device, Path('/dev/dri').is_dir())
            if Path('/dev/dri').is_dir():
                self.assertIn('--device=/dev/dri:/dev/dri', args)
            self.assertIn('--network', args)
            self.assertIn('bridge', args)
            self.assertNotIn('--privileged', args)
            self.assertIn('29900:19900', args)
            self.assertFalse(any(':19901' in arg for arg in args))


if __name__ == '__main__':
    unittest.main()
