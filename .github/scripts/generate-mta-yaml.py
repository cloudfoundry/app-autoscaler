#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = ["pyyaml"]
# ///
import re
import sys
import yaml

version = sys.argv[1]

with open("mta.tpl.yaml") as f:
    raw = f.read()

# Strip comment lines before parsing
raw = re.sub(r"^# .*\n", "", raw, flags=re.MULTILINE)
raw = raw.replace("MTA_VERSION", version)

data = yaml.safe_load(raw)

with open("mta.yaml", "w") as f:
    yaml.dump(data, f, default_flow_style=False, allow_unicode=True, sort_keys=False, width=10000)
