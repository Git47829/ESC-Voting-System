"""Tests for WebSocket endpoints /ws/votes and /ws/stats."""

import asyncio

import pytest

import main as app_module
from conftest import make_vote


@pytest.fixture(autouse=True)
def instant_keepalive(monkeypatch):
    """Interrupt long sleeps so keepalive loops exit quickly in tests."""
    real_sleep = asyncio.sleep

    async def fast_sleep(delay: float) -> None:
        if delay >= 1:
            raise asyncio.CancelledError
        await real_sleep(delay)

    monkeypatch.setattr("main.asyncio.sleep", fast_sleep)


def test_votes_ws_connects_successfully(client):
    with client.websocket_connect("/ws/votes"):
        pass


def test_votes_ws_sends_snapshot_when_df_populated(client):
    asyncio.run(app_module._handle_vote_message(make_vote(song_id=1, count=5)))
    assert len(app_module._vote_df) == 1

    with client.websocket_connect("/ws/votes") as ws:
        data = ws.receive_json()

    assert data["type"] == "snapshot"
    assert len(data["data"]) == 1
    assert data["data"][0]["song_id"] == 1


def test_votes_ws_no_snapshot_when_df_empty(client):
    assert app_module._vote_df.empty
    with client.websocket_connect("/ws/votes"):
        pass


def test_stats_ws_sends_initial_stats_payload(client):
    with client.websocket_connect("/ws/stats") as ws:
        data = ws.receive_json()

    assert data["type"] == "stats"
    assert data["vote_count"] == 0
    assert data["charts"] is None
