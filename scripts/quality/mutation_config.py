import json
from pathlib import Path


def frontend_config(targets: list[str], report: Path, vitest_config: str | None = None) -> str:
    vitest = "" if vitest_config is None else f'  vitest: {{ configFile: {json.dumps(vitest_config)} }},\n'
    return f"""export default {{
  testRunner: \"vitest\",
{vitest}  mutate: {json.dumps(targets)},
  reporters: [\"clear-text\", \"json\"],
  jsonReporter: {{ fileName: {json.dumps(str(report))} }},
  thresholds: {{ high: 100, low: 100, break: 100 }},
  timeoutMS: 10000,
  concurrency: 2,
}};\n"""
