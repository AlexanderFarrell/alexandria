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


@pytest.fixture(scope="session")
def registered_user(client: httpx.Client) -> dict:
    """Registers a single shared user for the test session.

    Uses a fixed username so the session is idempotent: if a previous run left
    the user in the database the fixture logs in instead of re-registering.
    Works with both REGISTRATION_MODE=single and REGISTRATION_MODE=multi.
    """
    username = "e2e_test_user"
    password = "testpass123"

    r = client.post("/api/v1/auth/register", json={"username": username, "password": password})
    if r.status_code == 201:
        data = r.json()
        return {"username": username, "password": password, "user": data["user"], "tokens": data["tokens"]}

    # User already exists (single-mode or re-run) — fall back to login.
    r = client.post("/api/v1/auth/login", json={"username": username, "password": password})
    assert r.status_code == 200, f"login fallback failed: {r.text}"
    data = r.json()
    return {"username": username, "password": password, "user": data["user"], "tokens": data["tokens"]}


@pytest.fixture(scope="session")
def auth_client(client: httpx.Client, registered_user: dict) -> httpx.Client:
    """An httpx Client pre-configured with the registered user's access token."""
    token = registered_user["tokens"]["access_token"]
    with httpx.Client(
        base_url=BASE_URL,
        timeout=30,
        headers={"Authorization": f"Bearer {token}"},
    ) as c:
        yield c
