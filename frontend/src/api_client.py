import time

import requests
from flask import current_app

from config import API_BASE, API_TIMEOUT
from telemetry import record_backend_call


def api_get(endpoint):
    """GET the backend API, recording call duration."""
    t0 = time.perf_counter()
    status_code = 500
    try:
        resp = requests.get(f"{API_BASE}{endpoint}", timeout=API_TIMEOUT)
        status_code = resp.status_code
        resp.raise_for_status()
        return resp.json()
    except requests.exceptions.RequestException as e:
        current_app.logger.error(
            "backend GET failed",
            extra={"api.endpoint": endpoint, "error": str(e)},
        )
        return None
    finally:
        record_backend_call(endpoint, status_code, time.perf_counter() - t0)


def api_get_auth(endpoint, params=None):
    """GET the backend API with query params, returning (status_code, json_body).

    Unlike api_get this never raises on non-2xx responses, making it suitable
    for authentication endpoints where the status code carries meaning.
    """
    t0 = time.perf_counter()
    status_code = 500
    try:
        resp = requests.get(
            f"{API_BASE}{endpoint}",
            params=params or {},
            timeout=API_TIMEOUT,
        )
        status_code = resp.status_code
        return resp.status_code, resp.json()
    except requests.exceptions.RequestException as e:
        current_app.logger.error(
            "backend auth GET failed",
            extra={"api.endpoint": endpoint, "error": str(e)},
        )
        return status_code, {"error": "Backend authentication service is currently unavailable. Please try again later."}
    finally:
        record_backend_call(endpoint, status_code, time.perf_counter() - t0)


def api_post(endpoint, params=None, cookies=None):
    """POST the backend API, recording call duration."""
    t0 = time.perf_counter()
    status_code = 500
    try:
        resp = requests.post(
            f"{API_BASE}{endpoint}",
            params=params or {},
            cookies=cookies or {},
            timeout=API_TIMEOUT,
        )
        status_code = resp.status_code
        return resp.status_code, resp.json(), resp.cookies
    except requests.exceptions.RequestException as e:
        current_app.logger.error(
            "backend POST failed",
            extra={"api.endpoint": endpoint, "error": str(e)},
        )
        return 502, {"error": "Backend service is currently unavailable. Please try again later."}, {}
    finally:
        record_backend_call(endpoint, status_code, time.perf_counter() - t0)


def api_delete(endpoint, params=None):
    """DELETE the backend API, recording call duration."""
    t0 = time.perf_counter()
    status_code = 500
    try:
        resp = requests.delete(
            f"{API_BASE}{endpoint}",
            params=params or {},
            timeout=API_TIMEOUT,
        )
        status_code = resp.status_code
        return resp.status_code, resp.json()
    except requests.exceptions.RequestException as e:
        current_app.logger.error(
            "backend DELETE failed",
            extra={"api.endpoint": endpoint, "error": str(e)},
        )
        return 502, {"error": "Backend service is currently unavailable. Please try again later."}
    finally:
        record_backend_call(endpoint, status_code, time.perf_counter() - t0)
