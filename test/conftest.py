"""Pytest configuration and shared fixtures for Alexandria e2e tests.

Assumes the server is running at BASE_URL (default http://localhost:8080).
Start the server before running tests, e.g.:
    make dev-server &
    cd test && pytest -v
"""

import os
import uuid

import httpx
import pytest

BASE_URL = os.getenv("ALEXANDRIA_URL", "http://localhost:8080")


@pytest.fixture(scope="session")
def base_url() -> str:
    return BASE_URL


@pytest.fixture(scope="session")
def client() -> httpx.Client:
    with httpx.Client(base_url=BASE_URL, timeout=10) as c:
        yield c


@pytest.fixture
def unique_user() -> dict:
    """Returns a unique username/password pair for test isolation."""
    suffix = uuid.uuid4().hex[:8]
    return {"username": f"test_{suffix}", "password": "testpass123"}


@pytest.fixture
def registered_user(client: httpx.Client, unique_user: dict) -> dict:
    """Registers a fresh user and returns user data + tokens."""
    r = client.post("/api/v1/auth/register", json=unique_user)
    assert r.status_code == 201, r.text
    data = r.json()
    return {
        "username": unique_user["username"],
        "password": unique_user["password"],
        "user": data["user"],
        "tokens": data["tokens"],
    }


@pytest.fixture
def auth_client(client: httpx.Client, registered_user: dict) -> httpx.Client:
    """An httpx Client pre-configured with the registered user's access token."""
    token = registered_user["tokens"]["access_token"]
    return httpx.Client(
        base_url=BASE_URL,
        timeout=10,
        headers={"Authorization": f"Bearer {token}"},
    )
