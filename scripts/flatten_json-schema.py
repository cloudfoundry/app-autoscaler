#! /usr/bin/env python3

import argparse
import json
import jsonref
from pathlib import Path
import sys


def merge_schemas(input_file_path):
    input_file_path_abs = Path(input_file_path).absolute()

    with open(input_file_path_abs, 'r') as f:
        policy_schema = jsonref.load(f, base_uri=input_file_path_abs.as_uri())

    json.dump(policy_schema, sys.stdout, indent=2)
    return None


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Flatten JSON schema by resolving references')
    parser.add_argument('input_file_path', help='Path to the input JSON schema file')
    args = parser.parse_args()

    merge_schemas(args.input_file_path)
