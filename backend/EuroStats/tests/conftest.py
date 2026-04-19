"""Shared pytest fixtures for EuroStats tests."""

import pandas as pd
import pytest
from starlette.testclient import TestClient

import main as app_module


@pytest.fixture(autouse=True)
def reset_vote_df():
    """Clear in-memory vote state before and after every test."""
    app_module._vote_df = pd.DataFrame(
        columns=[
            "song_id",
            "song_name",
            "country_voted_for",
            "country_voted_for_name",
            "voter_country",
            "voter_country_name",
            "vote_count",
            "timestamp",
        ]
    )
    yield
    app_module._vote_df = pd.DataFrame(
        columns=[
            "song_id",
            "song_name",
            "country_voted_for",
            "country_voted_for_name",
            "voter_country",
            "voter_country_name",
            "vote_count",
            "timestamp",
        ]
    )


@pytest.fixture
def client():
    """Return a TestClient without entering lifespan context managers."""
    return TestClient(app_module.app)


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
    return {
        "song_id": song_id,
        "song_name": song_name,
        "country_voted_for": country,
        "country_voted_for_name": country_name,
        "voter_country": voter,
        "voter_country_name": voter_name,
        "vote_count": count,
        "timestamp": ts,
    }
