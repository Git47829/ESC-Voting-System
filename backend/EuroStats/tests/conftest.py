"""
Shared pytest fixtures for EuroStats tests.

The FastAPI app uses a lifespan that tries to connect to a gRPC server on
startup. In tests we skip that by NOT using TestClient as a context manager
(which would trigger the lifespan). Instead we call TestClient() directly.
If the lifespan does run (Starlette decides to), it will fail gracefully
because the try/except swallows the connection error, leaving vote_consumer=None.
"""

from types import SimpleNamespace

import pandas as pd
import pytest
from starlette.testclient import TestClient

import main as app_module


# ---------------------------------------------------------------------------
# DataFrame reset — prevents state leakage between tests
# ---------------------------------------------------------------------------

@pytest.fixture(autouse=True)
def reset_vote_df():
    """Clear the in-memory vote DataFrame before and after every test."""
    app_module._vote_df = pd.DataFrame(columns=[
        "song_id", "song_name",
        "country_voted_for", "country_voted_for_name",
        "voter_country", "voter_country_name",
        "vote_count", "timestamp",
    ])
    yield
    app_module._vote_df = pd.DataFrame(columns=[
        "song_id", "song_name",
        "country_voted_for", "country_voted_for_name",
        "voter_country", "voter_country_name",
        "vote_count", "timestamp",
    ])


# ---------------------------------------------------------------------------
# vote_consumer reset — keeps it None so endpoints that require it return 503
# ---------------------------------------------------------------------------

@pytest.fixture(autouse=True)
def reset_vote_consumer():
    """Ensure vote_consumer is None so the lifespan is not needed."""
    original = app_module.vote_consumer
    app_module.vote_consumer = None
    yield
    app_module.vote_consumer = original


# ---------------------------------------------------------------------------
# TestClient (no lifespan startup)
# ---------------------------------------------------------------------------

@pytest.fixture
def client():
    """Return a TestClient that does not run the lifespan event handlers."""
    return TestClient(app_module.app)


# ---------------------------------------------------------------------------
# Sample vote helper
# ---------------------------------------------------------------------------

def make_vote(
    song_id=1,
    song_name="Test Song",
    country="DE",
    country_name="Germany",
    voter="FR",
    voter_name="France",
    count=5,
    ts=1700000000,
):
    return SimpleNamespace(
        song_id=song_id,
        song_name=song_name,
        country_voted_for=country,
        country_voted_for_name=country_name,
        voter_country=voter,
        voter_country_name=voter_name,
        vote_count=count,
        timestamp=ts,
    )
