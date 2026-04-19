"""Tests for GET /votes/subscribe endpoint."""

import asyncio

import main as app_module
from conftest import make_vote


def test_subscribe_returns_empty_payload_when_no_votes(client):
    response = client.get("/votes/subscribe")
    assert response.status_code == 200
    assert response.json() == {"votes": [], "count": 0}


def test_subscribe_returns_accumulated_votes(client):
    asyncio.run(app_module._handle_vote_message(make_vote(song_id=1, count=5)))
    asyncio.run(
        app_module._handle_vote_message(
            make_vote(song_id=2, country="SE", country_name="Sweden", count=10)
        )
    )

    response = client.get("/votes/subscribe")
    body = response.json()

    assert response.status_code == 200
    assert body["count"] == 2
    assert len(body["votes"]) == 2


def test_subscribe_response_shape(client):
    asyncio.run(app_module._handle_vote_message(make_vote()))

    response = client.get("/votes/subscribe")
    vote = response.json()["votes"][0]

    expected_keys = {
        "song_id",
        "song_name",
        "country_voted_for",
        "country_voted_for_name",
        "voter_country",
        "voter_country_name",
        "vote_count",
        "timestamp",
    }
    assert expected_keys == set(vote.keys())


def test_subscribe_ignores_legacy_query_params(client):
    asyncio.run(app_module._handle_vote_message(make_vote(song_id=10)))

    response = client.get("/votes/subscribe?include_historical=false")

    assert response.status_code == 200
    assert response.json()["count"] == 1
