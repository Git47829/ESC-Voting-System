"""Tests for the pure in-memory vote logic: handle_vote and _compute_and_broadcast."""

import asyncio
from unittest.mock import AsyncMock, patch

import pandas as pd
import pytest

import main as app_module
from conftest import make_vote


# ---------------------------------------------------------------------------
# handle_vote — DataFrame mutation
# ---------------------------------------------------------------------------

def test_handle_vote_appends_to_df():
    """Calling handle_vote should add one row to _vote_df."""
    v = make_vote(song_id=1, count=5)
    asyncio.run(app_module.handle_vote(v))
    assert len(app_module._vote_df) == 1
    assert app_module._vote_df.iloc[0]["song_id"] == 1
    assert app_module._vote_df.iloc[0]["vote_count"] == 5


def test_handle_vote_accumulates_multiple():
    """Multiple calls should accumulate rows, not replace them."""
    asyncio.run(app_module.handle_vote(make_vote(song_id=1, count=5)))
    asyncio.run(app_module.handle_vote(make_vote(song_id=2, count=10)))
    asyncio.run(app_module.handle_vote(make_vote(song_id=3, count=3)))
    assert len(app_module._vote_df) == 3


def test_handle_vote_stores_correct_fields():
    """All fields from the vote object should be stored in the DataFrame."""
    v = make_vote(
        song_id=42,
        song_name="Satellite",
        country="DE",
        country_name="Germany",
        voter="FR",
        voter_name="France",
        count=7,
        ts=1700000001,
    )
    asyncio.run(app_module.handle_vote(v))
    row = app_module._vote_df.iloc[0]
    assert row["song_id"] == 42
    assert row["song_name"] == "Satellite"
    assert row["country_voted_for"] == "DE"
    assert row["country_voted_for_name"] == "Germany"
    assert row["voter_country"] == "FR"
    assert row["voter_country_name"] == "France"
    assert row["vote_count"] == 7


# ---------------------------------------------------------------------------
# _compute_and_broadcast
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_compute_broadcast_skips_when_no_connections():
    """`_compute_and_broadcast` should do nothing when no WS clients are connected."""
    app_module.stats_manager.active_connections = []
    broadcast_mock = AsyncMock()
    with patch.object(app_module.stats_manager, "broadcast", broadcast_mock):
        await app_module._compute_and_broadcast()
    broadcast_mock.assert_not_called()


@pytest.mark.asyncio
async def test_compute_broadcast_empty_df_sends_zero_count():
    """With no votes, broadcast should receive vote_count=0."""
    # Add a dummy connection so broadcast isn't skipped.
    dummy_conn = object()
    app_module.stats_manager.active_connections = [dummy_conn]

    broadcast_mock = AsyncMock()
    with patch.object(app_module.stats_manager, "broadcast", broadcast_mock):
        await app_module._compute_and_broadcast()

    app_module.stats_manager.active_connections = []

    broadcast_mock.assert_called_once()
    call_args = broadcast_mock.call_args[0][0]
    assert call_args["type"] == "stats"
    assert call_args["vote_count"] == 0
    assert call_args["charts"] is None


@pytest.mark.asyncio
async def test_compute_broadcast_with_votes_sends_charts():
    """When votes are present, broadcast should include chart data URLs."""
    # Populate DataFrame with two votes from different countries.
    await app_module.handle_vote(make_vote(
        song_id=1, voter="DE", voter_name="Germany", count=10
    ))
    await app_module.handle_vote(make_vote(
        song_id=1, voter="FR", voter_name="France", count=5
    ))

    dummy_conn = object()
    app_module.stats_manager.active_connections = [dummy_conn]

    broadcast_mock = AsyncMock()
    with patch.object(app_module.stats_manager, "broadcast", broadcast_mock):
        await app_module._compute_and_broadcast()

    app_module.stats_manager.active_connections = []

    broadcast_mock.assert_called_once()
    call_args = broadcast_mock.call_args[0][0]
    assert call_args["type"] == "stats"
    charts = call_args.get("charts")
    assert charts is not None
    assert "voters_by_country" in charts
    assert "votes_received_by_country" in charts


# ---------------------------------------------------------------------------
# _make_pie_chart
# ---------------------------------------------------------------------------

def test_make_pie_chart_returns_base64_data_url():
    """_make_pie_chart should return a PNG data URL."""
    result = app_module._make_pie_chart(["Germany", "France"], [10, 5], "Test Chart")
    assert result.startswith("data:image/png;base64,")
    assert len(result) > 50  # non-trivial content
