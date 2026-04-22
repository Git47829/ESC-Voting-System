"""Tests for vote message handling and stats broadcast logic."""

import asyncio
from unittest.mock import AsyncMock, patch

import pytest

import main as app_module
from conftest import make_vote


def test_handle_vote_message_appends_to_df():
    vote = make_vote(song_id=1, count=5)
    asyncio.run(app_module._handle_vote_message(vote))
    assert len(app_module._vote_df) == 1
    assert app_module._vote_df.iloc[0]["song_id"] == 1
    assert app_module._vote_df.iloc[0]["vote_count"] == 5


def test_handle_vote_message_accumulates_multiple():
    asyncio.run(app_module._handle_vote_message(make_vote(song_id=1, count=5)))
    asyncio.run(app_module._handle_vote_message(make_vote(song_id=2, count=10)))
    asyncio.run(app_module._handle_vote_message(make_vote(song_id=3, count=3)))
    assert len(app_module._vote_df) == 3


def test_handle_vote_message_stores_correct_fields():
    vote = make_vote(
        song_id=42,
        song_name="Satellite",
        country="DE",
        country_name="Germany",
        voter="FR",
        voter_name="France",
        count=7,
        ts=1700000001,
    )
    asyncio.run(app_module._handle_vote_message(vote))
    row = app_module._vote_df.iloc[0]
    assert row["song_id"] == 42
    assert row["song_name"] == "Satellite"
    assert row["country_voted_for"] == "DE"
    assert row["country_voted_for_name"] == "Germany"
    assert row["voter_country"] == "FR"
    assert row["voter_country_name"] == "France"
    assert row["vote_count"] == 7
    assert row["timestamp"] == 1700000001


@pytest.mark.asyncio
async def test_handle_vote_message_publishes_to_redis_when_available():
    redis_mock = AsyncMock()
    with patch.object(app_module, "redis_client", redis_mock), patch.object(
        app_module, "_compute_and_broadcast", AsyncMock()
    ):
        await app_module._handle_vote_message(make_vote(song_id=5, count=2))

    redis_mock.rpush.assert_awaited_once()
    args = redis_mock.rpush.await_args.args
    assert args[0] == "votes:all"


@pytest.mark.asyncio
async def test_compute_broadcast_skips_when_no_connections():
    app_module.stats_manager.active_connections = []
    broadcast_mock = AsyncMock()
    with patch.object(app_module.stats_manager, "broadcast", broadcast_mock):
        await app_module._compute_and_broadcast()
    broadcast_mock.assert_not_called()


@pytest.mark.asyncio
async def test_compute_broadcast_empty_df_sends_zero_count():
    app_module.stats_manager.active_connections = [object()]
    broadcast_mock = AsyncMock()
    with patch.object(app_module.stats_manager, "broadcast", broadcast_mock):
        await app_module._compute_and_broadcast()

    app_module.stats_manager.active_connections = []

    payload = broadcast_mock.call_args.args[0]
    assert payload["type"] == "stats"
    assert payload["vote_count"] == 0
    assert payload["charts"] is None


@pytest.mark.asyncio
async def test_compute_broadcast_with_votes_sends_charts():
    await app_module._handle_vote_message(make_vote(song_id=1, voter="DE", voter_name="Germany", count=10))
    await app_module._handle_vote_message(make_vote(song_id=1, voter="FR", voter_name="France", count=5))

    app_module.stats_manager.active_connections = [object()]
    broadcast_mock = AsyncMock()
    with patch.object(app_module.stats_manager, "broadcast", broadcast_mock):
        await app_module._compute_and_broadcast()

    app_module.stats_manager.active_connections = []

    payload = broadcast_mock.call_args.args[0]
    assert payload["type"] == "stats"
    assert payload["vote_count"] == 15
    assert payload["charts"]["voters_by_country"].startswith("data:image/png;base64,")
    assert payload["charts"]["votes_received_by_country"].startswith("data:image/png;base64,")


def test_make_pie_chart_returns_base64_data_url():
    result = app_module._make_pie_chart(["Germany", "France"], [10, 5], "Test Chart")
    assert result.startswith("data:image/png;base64,")
    assert len(result) > 50
