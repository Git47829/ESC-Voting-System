"""Tests for GET /votes/subscribe endpoint."""

from unittest.mock import AsyncMock, MagicMock, patch

import main as app_module
from conftest import make_vote


# ---------------------------------------------------------------------------
# Consumer not initialized
# ---------------------------------------------------------------------------

def test_subscribe_returns_503_when_consumer_none(client):
    """When vote_consumer is None, the endpoint should return 503."""
    assert app_module.vote_consumer is None
    response = client.get("/votes/subscribe")
    assert response.status_code == 503
    assert "error" in response.json()


# ---------------------------------------------------------------------------
# Successful subscription with mock consumer
# ---------------------------------------------------------------------------

async def _votes_generator(*votes):
    """Helper: async generator yielding the given vote objects."""
    for v in votes:
        yield v


def test_subscribe_returns_votes_list(client):
    """subscribe_to_votes yields votes → endpoint returns them as a list."""
    v1 = make_vote(song_id=1, count=5)
    v2 = make_vote(song_id=2, country="SE", country_name="Sweden", count=10)

    mock_consumer = MagicMock()
    mock_consumer.subscribe_to_votes = MagicMock(
        return_value=_votes_generator(v1, v2)
    )

    with patch.object(app_module, "vote_consumer", mock_consumer):
        response = client.get("/votes/subscribe")

    assert response.status_code == 200
    body = response.json()
    assert body["count"] == 2
    assert len(body["votes"]) == 2


def test_subscribe_caps_at_100_entries(client):
    """Endpoint should stop consuming after 100 votes."""

    async def many_votes():
        for i in range(200):
            yield make_vote(song_id=i)

    mock_consumer = MagicMock()
    mock_consumer.subscribe_to_votes = MagicMock(return_value=many_votes())

    with patch.object(app_module, "vote_consumer", mock_consumer):
        response = client.get("/votes/subscribe")

    body = response.json()
    assert len(body["votes"]) == 100
    assert body["count"] == 100


def test_subscribe_include_historical_true_by_default(client):
    """include_historical defaults to True."""
    captured = {}

    async def capturing_generator(include_historical=True):
        captured["include_historical"] = include_historical
        return
        yield  # make it an async generator

    mock_consumer = MagicMock()
    mock_consumer.subscribe_to_votes = MagicMock(side_effect=capturing_generator)

    with patch.object(app_module, "vote_consumer", mock_consumer):
        client.get("/votes/subscribe")

    assert captured.get("include_historical") is True


def test_subscribe_include_historical_false_when_queried(client):
    """include_historical=false passes False to the consumer."""
    captured = {}

    async def capturing_generator(include_historical=True):
        captured["include_historical"] = include_historical
        return
        yield

    mock_consumer = MagicMock()
    mock_consumer.subscribe_to_votes = MagicMock(side_effect=capturing_generator)

    with patch.object(app_module, "vote_consumer", mock_consumer):
        client.get("/votes/subscribe?include_historical=false")

    assert captured.get("include_historical") is False


def test_subscribe_grpc_error_returns_502(client):
    """If the consumer raises an exception, the endpoint returns 502."""
    async def failing_generator(include_historical=True):
        raise RuntimeError("gRPC connection lost")
        yield  # noqa: unreachable

    mock_consumer = MagicMock()
    mock_consumer.subscribe_to_votes = MagicMock(side_effect=failing_generator)

    with patch.object(app_module, "vote_consumer", mock_consumer):
        response = client.get("/votes/subscribe")

    assert response.status_code == 502
    assert "error" in response.json()


def test_subscribe_response_shape(client):
    """Each vote in the payload should have all expected fields."""
    v = make_vote()

    mock_consumer = MagicMock()
    mock_consumer.subscribe_to_votes = MagicMock(
        return_value=_votes_generator(v)
    )

    with patch.object(app_module, "vote_consumer", mock_consumer):
        response = client.get("/votes/subscribe")

    body = response.json()
    vote = body["votes"][0]

    expected_keys = {
        "song_id", "song_name",
        "country_voted_for", "country_voted_for_name",
        "voter_country", "voter_country_name",
        "vote_count", "timestamp",
    }
    assert expected_keys == set(vote.keys())
