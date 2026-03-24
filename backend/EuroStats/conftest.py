"""
Root-level conftest: stub out telemetry before any test file imports main.

setup_telemetry() creates BatchLogRecordProcessor / BatchSpanProcessor threads
that are NOT daemon threads and block pytest from exiting while they retry
connecting to otel-collector:4317 with exponential back-off. Replacing the
telemetry module with a no-op mock prevents those threads from being spawned.
"""
import sys
from unittest.mock import MagicMock

sys.modules.setdefault("telemetry", MagicMock())
