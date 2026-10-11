"""Check security report summaries and immutable scan target boundaries."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('security_scan', Path(__file__).parents[1] / 'security/scan.py')
scan = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scan)


class SecurityScanTest(unittest.TestCase):
    def test_mutable_or_malformed_image_is_rejected(self):
        for image in ['example/image:nightly', 'example/image@sha256:123', '--password=secret',
                      'https://user:secret@example/image@sha256:' + 'a' * 64]:
            with self.assertRaises(ValueError):
                scan.validate_image(image)

    def test_exact_digest_is_accepted(self):
        image = 'ghcr.io/example/image@sha256:' + 'a' * 64
        self.assertEqual(scan.validate_image(image), image)

    def test_summary_retains_unknown_and_unfixed_without_leaking_matches(self):
        report = {'Results': [{'Vulnerabilities': [{'Severity': 'HIGH', 'Description': 'private'},
                                                  {'Severity': 'UNKNOWN'}, {'Severity': 'HIGH'}],
                               'Secrets': [{'Match': 'sensitive'}]}, {}]}
        self.assertEqual(scan.counts(report), ({'HIGH': 2, 'UNKNOWN': 1}, 1))


if __name__ == '__main__':
    unittest.main()
