#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "beautifulsoup4>=4.14.3",
#     "packaging>=26.0",
#     "requests>=2.33.1",
# ]
# ///
"""Check if the OpenTelemetry SDK version in go.mod is supported by Dynatrace OneAgent."""

import re
import sys
from pathlib import Path

import requests
from bs4 import BeautifulSoup
from packaging.version import Version


DOCS_URL = (
    "https://docs.dynatrace.com/docs/ingest-from/"
    "dynatrace-oneagent/oneagent-and-opentelemetry/configuration"
)

GO_MOD_OTEL_SDK_PATTERN = re.compile(
    r"go\.opentelemetry\.io/otel/sdk\s+v(\S+)"
)

VERSION_RANGE_PATTERN = re.compile(
    r"^(?P<low>\d+(?:\.\d+)*)\s*-\s*(?P<high>\d+(?:\.\d+)*)$"
)


def get_otel_sdk_version(go_mod_path: str) -> str:
    """Extract the go.opentelemetry.io/otel/sdk version from go.mod."""
    text = Path(go_mod_path).read_text()
    match = GO_MOD_OTEL_SDK_PATTERN.search(text)
    if not match:
        print("::error::go.opentelemetry.io/otel/sdk not found in go.mod")
        sys.exit(1)
    return match.group(1)


def get_supported_version_ranges() -> list[str]:
    """Scrape supported OTel version ranges from Dynatrace docs."""
    resp = requests.get(DOCS_URL, timeout=30)
    resp.raise_for_status()

    soup = BeautifulSoup(resp.text, "html.parser")

    go_panel = soup.find(attrs={"aria-labelledby": "prereq--go"})
    if not go_panel:
        print("::error::Go tab panel not found on the Dynatrace docs page")
        sys.exit(1)

    table = go_panel.find("table")
    if not table:
        print("::error::Version table not found in Go tab panel")
        sys.exit(1)

    version_spans = table.select(
        'span[aria-label="Show Dynatrace OneAgent version support info"]'
    )
    versions = []
    for span in version_spans:
        text_el = span.find(attrs={"data-dt-component": "Text"})
        if text_el:
            versions.append(text_el.get_text(strip=True))

    if not versions:
        print("::error::No version ranges found in the Dynatrace docs")
        sys.exit(1)

    return versions


def is_version_in_range(version_str: str, range_str: str) -> bool:
    """Check if a version falls within a given 'low - high' range."""
    match = VERSION_RANGE_PATTERN.match(range_str.strip())
    if not match:
        return False

    ver = Version(version_str)
    low = Version(match.group("low"))
    high = Version(match.group("high"))
    return low <= ver <= high


def main() -> None:
    go_mod_path = sys.argv[1] if len(sys.argv) > 1 else "go.mod"

    sdk_version = get_otel_sdk_version(go_mod_path)
    print(f"OTel SDK version in go.mod: {sdk_version}")

    supported_ranges = get_supported_version_ranges()
    print(f"Supported version ranges from Dynatrace docs: {supported_ranges}")

    for version_range in supported_ranges:
        if is_version_in_range(sdk_version, version_range):
            print(
                f"Version {sdk_version} is within supported range"
                f" '{version_range}'."
            )
            return

    # Not supported — output for GitHub Actions
    ranges_formatted = ", ".join(supported_ranges)
    message = (
        f"The OpenTelemetry SDK version `v{sdk_version}` in `go.mod` is"
        f" **not supported** by Dynatrace OneAgent.\n\n"
        f"Supported version ranges (from [Dynatrace docs]({DOCS_URL}#prereq--go)):\n"
    )
    for r in supported_ranges:
        message += f"- `{r}`\n"
    message += (
        "\nPlease verify Dynatrace OneAgent compatibility before merging."
    )

    print(f"\n::error::OTel SDK v{sdk_version} is not in supported ranges: {ranges_formatted}")

    # Write outputs for the workflow
    import os

    github_output = os.environ.get("GITHUB_OUTPUT", "")
    if github_output:
        with open(github_output, "a") as f:
            # Use multiline syntax for the comment body
            f.write("supported=false\n")
            f.write(f"sdk_version={sdk_version}\n")
            delimiter = "EOF_COMMENT_BODY"
            f.write(f"comment_body<<{delimiter}\n")
            f.write(message)
            f.write(f"\n{delimiter}\n")

    sys.exit(1)


if __name__ == "__main__":
    main()
