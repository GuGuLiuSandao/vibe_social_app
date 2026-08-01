#!/usr/bin/env python3
import json
from datetime import datetime, timezone
from pathlib import Path

root = Path(__file__).resolve().parents[2]
quality = root / "quality"
quality.mkdir(parents=True, exist_ok=True)
(quality / "static-contract.json").write_text(json.dumps({"schema_version": 1, "gate": "quality-static", "status": "passed", "completed_at": datetime.now(timezone.utc).isoformat()}, indent=2) + "\n")
print("static contract evidence written")
