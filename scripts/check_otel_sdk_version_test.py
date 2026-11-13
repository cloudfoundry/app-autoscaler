"""Offline regression tests for the Dynatrace documentation parser."""

import unittest

from check_otel_sdk_version import is_version_in_range, parse_supported_version_ranges


def version_table(version_range: str) -> str:
    return (
        '<table><span aria-label="Show Dynatrace OneAgent version support info">'
        f'<span data-dt-component="Text">{version_range}</span></span></table>'
    )


class SupportedVersionTests(unittest.TestCase):
    def test_tabbed_docs(self):
        html = '<div aria-labelledby="prereq--go">' + version_table("1.28 - 1.47") + '</div>'
        self.assertEqual(parse_supported_version_ranges(html), ["1.28 - 1.47"])

    def test_heading_docs_exclude_other_languages(self):
        html = (
            '<h3 id="prereq-java">Java</h3>' + version_table("1.0 - 1.65")
            + '<h3 id="prereq-go">Go</h3><div>' + version_table("1.28 - 1.47") + '</div>'
            + '<h3 id="prereq-nodejs">Node.js</h3>' + version_table("2.0 - 2.8")
        )
        self.assertEqual(parse_supported_version_ranges(html), ["1.28 - 1.47"])

    def test_missing_go_table_fails_closed(self):
        html = '<h3 id="prereq-go">Go</h3><h3>Java</h3>' + version_table("1.0 - 1.65")
        with self.assertRaises(ValueError):
            parse_supported_version_ranges(html)

    def test_missing_go_section_fails_closed(self):
        with self.assertRaises(ValueError):
            parse_supported_version_ranges('<h3>Java</h3>' + version_table("1.0 - 1.65"))

    def test_missing_version_spans_fails_closed(self):
        with self.assertRaises(ValueError):
            parse_supported_version_ranges('<h3 id="prereq-go">Go</h3><table></table>')

    def test_range_boundaries(self):
        self.assertTrue(is_version_in_range("1.47.0", "1.28 - 1.47"))
        self.assertFalse(is_version_in_range("1.48.0", "1.28 - 1.47"))


if __name__ == "__main__":
    unittest.main()
