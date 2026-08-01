import json
from pathlib import Path


def frontend_config(targets: list[str], report: Path) -> str:
    return f"""export default {{
  testRunner: \"vitest\",
  mutate: {json.dumps(targets)},
  reporters: [\"clear-text\", \"json\"],
  jsonReporter: {{ fileName: {json.dumps(str(report))} }},
  thresholds: {{ high: 100, low: 100, break: 100 }},
  timeoutMS: 10000,
  concurrency: 2,
}};\n"""
