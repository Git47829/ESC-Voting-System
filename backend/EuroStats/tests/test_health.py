"""Tests for GET /health endpoint."""


def test_health_returns_200(client):
    response = client.get("/health")
    assert response.status_code == 200


def test_health_body_is_healthy(client):
    response = client.get("/health")
    body = response.json()
    assert body == {"status": "healthy"}


def test_health_content_type_json(client):
    response = client.get("/health")
    assert "application/json" in response.headers.get("content-type", "")
