#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = ["pyyaml"]
# ///
import re
import sys
import yaml

version = sys.argv[1]
url_base = (
    f"https://github.com/cloudfoundry/app-autoscaler/releases/download/"
    f"v{version}/app-autoscaler-release-v{version}.mtar"
)
new_cmd = (
    f'bash -c "curl -fsSL -o /tmp/upstream.mtar {url_base}'
    " && python3 -c \\\"import zipfile; zipfile.ZipFile('/tmp/upstream.mtar')"
    ".extract('acceptance-tests/build/acceptance/data.zip', '/tmp/upstream-mtar')\\\""
    " && mkdir -p build/acceptance"
    " && python3 -c \\\"import zipfile; zipfile.ZipFile('/tmp/upstream-mtar"
    "/acceptance-tests/build/acceptance/data.zip').extractall('build/acceptance/')\\\"\""
)

with open("mta.tpl.yaml") as f:
    raw = f.read()

# Strip comment lines before parsing
raw = re.sub(r"^# .*\n", "", raw, flags=re.MULTILINE)
raw = raw.replace("MTA_VERSION", version)

data = yaml.safe_load(raw)

for module in data.get("modules", []):
    name = module.get("name")
    if name == "acceptance-tests":
        module["build-parameters"]["commands"] = [new_cmd]

with open("mta.yaml", "w") as f:
    yaml.dump(data, f, default_flow_style=False, allow_unicode=True, sort_keys=False, width=10000)
