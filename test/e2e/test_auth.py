"""E2E tests for authentication endpoints."""

import uuid
import httpx
import pytest


def test_health(client: httpx.Client):
    r = client.get("/health")
    assert r.status_code == 200
    assert r.json()["status"] == "ok"


def test_register_success(client: httpx.Client):
    user = {"username": f"user_{uuid.uuid4().hex[:8]}", "password": "securepass1"}
    r = client.post("/api/v1/auth/register", json=user)
    assert r.status_code == 201
    data = r.json()
    assert "user" in data
    assert data["user"]["username"] == user["username"]
    assert "tokens" in data
    assert "access_token" in data["tokens"]
    assert "refresh_token" in data["tokens"]


def test_register_duplicate_username(client: httpx.Client, registered_user: dict):
    r = client.post(
        "/api/v1/auth/register",
        json={"username": registered_user["username"], "password": "newpass123"},
    )
    assert r.status_code == 409


def test_register_short_password(client: httpx.Client):
    r = client.post(
        "/api/v1/auth/register",
        json={"username": f"u_{uuid.uuid4().hex[:8]}", "password": "short"},
    )
    assert r.status_code == 400


def test_login_success(client: httpx.Client, registered_user: dict):
    r = client.post(
        "/api/v1/auth/login",
        json={"username": registered_user["username"], "password": registered_user["password"]},
    )
    assert r.status_code == 200
    data = r.json()
    assert "tokens" in data
    assert "access_token" in data["tokens"]


def test_login_wrong_password(client: httpx.Client, registered_user: dict):
    r = client.post(
        "/api/v1/auth/login",
        json={"username": registered_user["username"], "password": "wrongpassword"},
    )
    assert r.status_code == 401


def test_login_unknown_user(client: httpx.Client):
    r = client.post(
        "/api/v1/auth/login",
        json={"username": "nobody_ever", "password": "whatever123"},
    )
    assert r.status_code == 401


def test_refresh_token(client: httpx.Client, registered_user: dict):
    r = client.post(
        "/api/v1/auth/refresh",
        json={"refresh_token": registered_user["tokens"]["refresh_token"]},
    )
    assert r.status_code == 200
    assert "access_token" in r.json()["tokens"]


def test_protected_route_without_token(client: httpx.Client):
    r = client.get("/api/v1/books")
    assert r.status_code == 401


def test_me(auth_client: httpx.Client, registered_user: dict):
    r = auth_client.get("/api/v1/me")
    assert r.status_code == 200
    assert "user_id" in r.json()
