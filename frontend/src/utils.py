import json
import re
from functools import wraps
from urllib.parse import unquote

from flask import abort, redirect, session, url_for

from api_client import api_get


def normalize_youtube_url(url: str) -> str:
    """Convert any YouTube URL format to the embeddable /embed/VIDEO_ID form.

    Handles:
      - https://www.youtube.com/watch?v=VIDEO_ID
      - https://youtu.be/VIDEO_ID
      - https://www.youtube.com/embed/VIDEO_ID   (already correct)
      - https://youtube.com/shorts/VIDEO_ID
    Returns the original string unchanged if no video ID can be extracted.
    """
    if not url:
        return url

    # Already an embed URL — return as-is
    embed_match = re.search(r"youtube\.com/embed/([A-Za-z0-9_-]{11})", url)
    if embed_match:
        return f"https://www.youtube.com/embed/{embed_match.group(1)}"

    # youtu.be/VIDEO_ID
    short_match = re.search(r"youtu\.be/([A-Za-z0-9_-]{11})", url)
    if short_match:
        return f"https://www.youtube.com/embed/{short_match.group(1)}"

    # youtube.com/watch?v=VIDEO_ID  or  youtube.com/shorts/VIDEO_ID
    watch_match = re.search(
        r"youtube\.com/(?:watch\?(?:.*&)?v=|shorts/)([A-Za-z0-9_-]{11})", url
    )
    if watch_match:
        return f"https://www.youtube.com/embed/{watch_match.group(1)}"

    return url


def decode_vote_state_cookie(cookie_value):
    """
    Best-effort decode of the vote_state cookie written by the Go backend.
    The cookie is: hex( JSON . hex(HMAC-SHA256) )
    We only need the JSON payload for display purposes — signature verification
    happens authoritatively in the Go API on every POST, so we just try to
    parse the inner JSON and fall back gracefully on any error.
    """
    try:
        raw = bytes.fromhex(cookie_value)
        # Find the last '.' which separates JSON from the hex signature
        sep = raw.rfind(b".")
        if sep == -1:
            return None
        payload_bytes = raw[:sep]
        return json.loads(payload_bytes.decode("utf-8"))
    except Exception:
        return None


def decode_consent_cookie(cookie_value):
    """Decode the esc_cookie_consent JSON cookie written by frontend JS."""
    if not cookie_value:
        return None
    try:
        return json.loads(unquote(cookie_value))
    except Exception:
        return None


def get_voting_status():
    """Fetch the current voting open/closed status from the songs endpoint."""
    data = api_get("/songs/")
    if data and "payload" in data and len(data["payload"]) > 0:
        return data["payload"][0].get("votingIsOpen", False)
    return False


def login_required(role):
    """Decorator that enforces a session role. Admins can access jury routes."""

    def decorator(f):
        @wraps(f)
        def decorated_function(*args, **kwargs):
            if "role" not in session or "token" not in session:
                return redirect(url_for("auth.login"))
            if session["role"] != role and session["role"] != "admin":
                abort(403)
            return f(*args, **kwargs)

        return decorated_function

    return decorator
