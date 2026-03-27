import json
import os
import re
import time
from functools import wraps

import requests
from flask import (
    Flask,
    abort,
    flash,
    jsonify,
    make_response,
    redirect,
    render_template,
    request,
    session,
    url_for,
)

app = Flask(__name__)
app.secret_key = os.environ.get("FLASK_SECRET_KEY", "esc-voting-secret-key-change-me")

API_BASE = os.environ.get("API_BASE_URL", "http://db-crud-api:8000")
API_TIMEOUT = int(os.environ.get("API_TIMEOUT", "10"))
ESC_CONVERTER_URL = os.environ.get(
    "ESC_CONVERTER_URL", "http://public-vote-converter:8090"
)
EUROSTATS_URL = os.environ.get("EUROSTATS_URL", "http://eurostats:8880")

# ---------------------------------------------------------------------------
# Telemetry — initialised immediately after the app object is created, before
# any helpers or routes, so that FlaskInstrumentor wraps all route registrations
# and RequestsInstrumentor patches the requests library before first use.
# ---------------------------------------------------------------------------
from telemetry import (  # noqa: E402 — intentional post-app import
    record_backend_call,
    record_vote,
    set_active_sessions,
    setup_telemetry,
)

setup_telemetry(app)


# ---------------------------------------------------------------------------
# Backend API helpers
# ---------------------------------------------------------------------------


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
        app.logger.error(
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
        app.logger.error(
            "backend auth GET failed",
            extra={"api.endpoint": endpoint, "error": str(e)},
        )
        return 502, {"error": str(e)}
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
        app.logger.error(
            "backend POST failed",
            extra={"api.endpoint": endpoint, "error": str(e)},
        )
        return 502, {"error": str(e)}, {}
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
        app.logger.error(
            "backend DELETE failed",
            extra={"api.endpoint": endpoint, "error": str(e)},
        )
        return 502, {"error": str(e)}
    finally:
        record_backend_call(endpoint, status_code, time.perf_counter() - t0)


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
                return redirect(url_for("login"))
            if session["role"] != role and session["role"] != "admin":
                abort(403)
            return f(*args, **kwargs)

        return decorated_function

    return decorator


# ---------------------------------------------------------------------------
# Context processor — inject voting status into every template render
# ---------------------------------------------------------------------------


@app.context_processor
def inject_voting_status():
    try:
        is_open = get_voting_status()
    except Exception:
        is_open = False
    return dict(voting_is_open=is_open)


# ---------------------------------------------------------------------------
# Public routes
# ---------------------------------------------------------------------------


TOTAL_VOTE_POINTS = 20


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


@app.route("/")
def vote_page():
    """Home / Vote page — shows country cards grid."""
    songs_data = api_get("/songs/")
    songs = []
    if songs_data and "payload" in songs_data:
        seen_ids = set()
        for s in songs_data["payload"]:
            if s["songId"] not in seen_ids:
                seen_ids.add(s["songId"])
                songs.append(s)

    vote_state = None
    raw_cookie = request.cookies.get("vote_state")
    if raw_cookie:
        vote_state = decode_vote_state_cookie(raw_cookie)

    if vote_state is not None:
        votes_remaining = vote_state.get("votes_remaining", 0)
        votes_cast = vote_state.get("votes_cast", {})
    else:
        votes_remaining = TOTAL_VOTE_POINTS
        votes_cast = {}

    out_of_points = votes_remaining == 0

    return render_template(
        "vote.html",
        songs=songs,
        votes_remaining=votes_remaining,
        votes_cast=votes_cast,
        out_of_points=out_of_points,
        total_vote_points=TOTAL_VOTE_POINTS,
    )


@app.route("/results")
def results_page():
    """Live results page — bar chart of votes, polled every 10 s."""
    return render_template("results.html")


@app.route("/stats")
def stats_page():
    """Live statistics page — pie charts from EuroStats WebSocket."""
    return render_template("stats.html")


@app.route("/api/results")
def api_results():
    """Combined jury + ESC-converted public ranking, computed on-the-fly."""
    t0 = time.perf_counter()
    esc_status = 500
    jury_status = 500
    try:
        esc_resp = requests.get(
            f"{ESC_CONVERTER_URL}/api/esc-points", timeout=API_TIMEOUT
        )
        esc_status = esc_resp.status_code
        esc_resp.raise_for_status()
        esc_data = esc_resp.json()
    except requests.exceptions.RequestException as e:
        app.logger.error("converter GET failed", extra={"error": str(e)})
        return jsonify([]), 503
    finally:
        record_backend_call("/api/esc-points", esc_status, time.perf_counter() - t0)

    jury_data = api_get("/votes/")
    jury_status = 200 if jury_data else 503

    # Build a map of song_id → jury points from the CRUD API response.
    jury_map = {}
    if jury_data and "payload" in jury_data:
        for entry in jury_data["payload"]:
            jury_map[entry["id"]] = entry.get("juryVotes", 0)

    # Merge ESC-converted public points with jury points.
    results = []
    if esc_data and "payload" in esc_data:
        for song in esc_data["payload"]:
            song_id = song["songId"]
            esc_pts = song.get("escPoints", 0)
            jury_pts = jury_map.get(song_id, 0)
            results.append(
                {
                    "id": song_id,
                    "name": song.get("songName", ""),
                    "country": song.get("country", ""),
                    "countryId": song.get("countryId", ""),
                    "escPublicPts": esc_pts,
                    "juryPts": jury_pts,
                    "totalPts": esc_pts + jury_pts,
                }
            )

    # Sort by combined total descending, assign rank.
    results.sort(key=lambda x: (-x["totalPts"], x["id"]))
    for i, entry in enumerate(results):
        entry["rank"] = i + 1

    return jsonify(results)


@app.route("/vote/submit", methods=["POST"])
def submit_vote():
    """Handle public vote form submission."""
    song_id = request.form.get("songID")
    phone = request.form.get("phoneNum")
    own_country = request.form.get("ownCountry", "")
    points = request.form.get("points", "1")

    if not song_id or not phone:
        return jsonify({"error": "Song and phone number are required"}), 422

    try:
        points_int = int(points)
        if points_int < 1 or points_int > TOTAL_VOTE_POINTS:
            return jsonify(
                {"error": f"Points must be between 1 and {TOTAL_VOTE_POINTS}"}
            ), 422
    except ValueError:
        return jsonify({"error": "Points must be an integer"}), 422

    # Forward the browser's vote_state cookie to the Go API
    browser_cookies = {}
    vote_state_cookie = request.cookies.get("vote_state")
    if vote_state_cookie:
        browser_cookies["vote_state"] = vote_state_cookie

    status, data, api_cookies = api_post(
        "/vote/",
        params={
            "songID": song_id,
            "phoneNum": phone,
            "ownCountry": own_country,
            "points": points,
        },
        cookies=browser_cookies,
    )

    if status == 200:
        record_vote("public")
        app.logger.info(
            "public vote cast",
            extra={"song_id": song_id, "own_country": own_country, "points": points},
        )

    flask_response = jsonify(data)
    flask_response.status_code = status

    # Forward the vote_state cookie from the Go API back to the browser
    if "vote_state" in api_cookies:
        flask_response.set_cookie(
            "vote_state",
            api_cookies["vote_state"],
            max_age=365 * 24 * 60 * 60,
            httponly=True,
            samesite="Strict",
        )

    return flask_response


# ---------------------------------------------------------------------------
# Auth routes
# ---------------------------------------------------------------------------


@app.route("/login", methods=["GET", "POST"])
def login():
    """Login page — token + role selector."""
    if request.method == "GET":
        return render_template("login.html")

    token = request.form.get("token", "").strip()
    role = request.form.get("role", "admin").strip()

    if not token:
        flash("Token is required.", "error")
        return render_template("login.html"), 401

    if role == "admin":
        status, data = api_get_auth("/admin/authenticate", params={"Token": token})
        if status != 202:
            app.logger.warning("failed admin login attempt")
            flash(data.get("error", "Invalid token."), "error")
            return render_template("login.html"), 401
        session["token"] = token
        session["role"] = "admin"
        set_active_sessions(1)
        app.logger.info("admin login", extra={"role": "admin"})
        return redirect(url_for("admin_dashboard"))
    elif role == "jury":
        status, data = api_get_auth("/jury/authenticate", params={"Token": token})
        if status != 202:
            app.logger.warning("failed jury login attempt")
            flash(data.get("error", "Invalid token."), "error")
            return render_template("login.html"), 401
        session["token"] = token
        session["role"] = "jury"
        set_active_sessions(1)
        app.logger.info("jury login", extra={"role": "jury"})
        return redirect(url_for("jury_page"))
    else:
        flash("Invalid role.", "error")
        return render_template("login.html"), 422


@app.route("/logout")
def logout():
    """Clear session and redirect to home."""
    role = session.get("role", "unknown")
    session.clear()
    set_active_sessions(0)
    app.logger.info("logout", extra={"role": role})
    return redirect(url_for("vote_page"))


# ---------------------------------------------------------------------------
# Admin routes
# ---------------------------------------------------------------------------


@app.route("/admin")
@login_required("admin")
def admin_dashboard():
    """Admin dashboard — voting controls, entries table, add-entry forms."""
    songs_data = api_get("/songs/")
    countries_data = api_get("/countries/")

    songs = []
    if songs_data and "payload" in songs_data:
        seen_ids = set()
        for s in songs_data["payload"]:
            if s["songId"] not in seen_ids:
                seen_ids.add(s["songId"])
                songs.append(s)

    countries = []
    if countries_data and "payload" in countries_data:
        countries = countries_data["payload"]

    return render_template("admin.html", songs=songs, countries=countries)


@app.route("/admin/open", methods=["POST"])
@login_required("admin")
def admin_open_vote():
    """Open voting."""
    token = session.get("token", "")
    status, data, _ = api_post("/admin/open/", params={"Token": token})
    if status in (200, 202):
        app.logger.info("voting opened by admin")
        flash("Voting has been opened!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to open voting.")), "error")
    return redirect(url_for("admin_dashboard"))


@app.route("/admin/close", methods=["POST"])
@login_required("admin")
def admin_close_vote():
    """Close voting."""
    token = session.get("token", "")
    status, data, _ = api_post("/admin/close", params={"Token": token})
    if status in (200, 202):
        app.logger.info("voting closed by admin")
        flash("Voting has been closed!", "success")
    else:
        flash(
            data.get("message", data.get("error", "Failed to close voting.")), "error"
        )
    return redirect(url_for("admin_dashboard"))


@app.route("/admin/reset", methods=["POST"])
@login_required("admin")
def admin_reset_votes():
    """Reset all votes to zero."""
    token = session.get("token", "")
    status, data = api_delete("/admin/deleteVotes/", params={"Token": token})
    if status in (200, 202):
        try:
            requests.post(f"{EUROSTATS_URL}/reset", timeout=API_TIMEOUT)
        except Exception as e:
            app.logger.warning(f"Failed to reset EuroStats in-memory state: {e}")
        app.logger.warning("all votes reset by admin")
        flash("All votes have been reset!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to reset votes.")), "error")
    return redirect(url_for("admin_dashboard"))


@app.route("/admin/add-country", methods=["POST"])
@login_required("admin")
def admin_add_country():
    """Add a new country."""
    token = session.get("token", "")
    country_id = request.form.get("id", "").strip()
    name = request.form.get("name", "").strip()
    pot = request.form.get("pot", "1").strip()

    if not country_id or not name:
        flash("Country ID and Name are required.", "error")
        return redirect(url_for("admin_dashboard"))

    status, data, _ = api_post(
        "/admin/addCountry/",
        params={"Token": token, "ID": country_id, "Name": name, "Pot": pot},
    )
    if status in (200, 202):
        app.logger.info(
            "country added", extra={"country_id": country_id, "country_name": name}
        )
        flash(f"Country '{name}' added successfully!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to add country.")), "error")
    return redirect(url_for("admin_dashboard"))


@app.route("/admin/add-artist", methods=["POST"])
@login_required("admin")
def admin_add_artist():
    """Add a new artist."""
    token = session.get("token", "")
    artist_id = request.form.get("id", "").strip()
    first_name = request.form.get("firstName", "").strip()
    last_name = request.form.get("lastName", "").strip()
    artist_type = request.form.get("type", "solo").strip()
    country = request.form.get("country", "").strip()

    if not artist_id or not last_name or not country:
        flash("Artist ID, Last Name, and Country are required.", "error")
        return redirect(url_for("admin_dashboard"))

    status, data, _ = api_post(
        "/admin/addArtist/",
        params={
            "Token": token,
            "ID": artist_id,
            "Name": last_name,
            "vorName": first_name,
            "typ": artist_type,
            "Land": country,
        },
    )
    if status in (200, 202):
        app.logger.info(
            "artist added",
            extra={"artist_id": artist_id, "artist_name": f"{first_name} {last_name}"},
        )
        flash(f"Artist '{first_name} {last_name}' added successfully!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to add artist.")), "error")
    return redirect(url_for("admin_dashboard"))


@app.route("/admin/add-song", methods=["POST"])
@login_required("admin")
def admin_add_song():
    """Add a new song."""
    token = session.get("token", "")
    song_name = request.form.get("name", "").strip()
    country = request.form.get("country", "").strip()
    artist_id = request.form.get("artistId", "").strip()
    youtube_url = request.form.get("youtubeUrl", "").strip()

    if not song_name or not country or not artist_id:
        flash("Song Name, Country, and Artist ID are required.", "error")
        return redirect(url_for("admin_dashboard"))

    params = {"Token": token, "ID": artist_id, "Name": song_name, "Land": country}
    if youtube_url:
        params["YoutubeURL"] = youtube_url

    status, data, _ = api_post("/admin/addSong/", params=params)
    if status in (200, 202):
        app.logger.info(
            "song added", extra={"song_name": song_name, "country": country}
        )
        flash(f"Song '{song_name}' added successfully!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to add song.")), "error")
    return redirect(url_for("admin_dashboard"))


@app.route("/admin/start-contest", methods=["POST"])
@login_required("admin")
def admin_start_contest():
    """Shuffle all songs into a random order and start the contest."""
    token = session.get("token", "")
    status, data, _ = api_post("/admin/startContest/", params={"Token": token})
    if status == 200:
        song_count = data.get("songCount", 0)
        app.logger.info("contest started", extra={"song_count": song_count})
        flash(f"Contest started! {song_count} songs queued in random order.", "success")
    else:
        flash(
            data.get("message", data.get("error", "Failed to start contest.")), "error"
        )
    return redirect(url_for("admin_dashboard"))


@app.route("/admin/advance-contest", methods=["POST"])
@login_required("admin")
def admin_advance_contest():
    """Advance the contest to the next song."""
    token = session.get("token", "")
    status, data, _ = api_post("/admin/advanceContest/", params={"Token": token})
    if status == 200:
        if data.get("finished"):
            flash("The contest has finished! All songs have performed.", "success")
        else:
            flash(
                f"Advanced to song {int(data.get('currentIndex', 0)) + 1}.", "success"
            )
    else:
        flash(
            data.get("message", data.get("error", "Failed to advance contest.")),
            "error",
        )
    return redirect(url_for("now_playing"))


@app.route("/now")
def now_playing():
    """Running Now page — shows the current song with YouTube embed and voting."""
    data = api_get("/contest/current/")
    if data is None or "error" in data:
        return render_template(
            "now.html",
            song=None,
            contest_active=False,
            countries=[],
            error=data.get("error", "No active contest")
            if data
            else "Could not reach backend",
        )

    song = data.get("payload")
    if song and song.get("youtubeUrl"):
        song["youtubeUrl"] = normalize_youtube_url(song["youtubeUrl"])

    # Fetch all songs so we can build the country selector server-side
    songs_data = api_get("/songs/")
    countries = []
    if songs_data and "payload" in songs_data:
        seen_ids = set()
        for s in songs_data["payload"]:
            if s["countryId"] not in seen_ids:
                seen_ids.add(s["countryId"])
                countries.append({"id": s["countryId"], "name": s["countryName"]})

    vote_state = None
    raw_cookie = request.cookies.get("vote_state")
    if raw_cookie:
        vote_state = decode_vote_state_cookie(raw_cookie)

    votes_remaining = (
        vote_state.get("votes_remaining", 0) if vote_state else TOTAL_VOTE_POINTS
    )
    votes_cast = vote_state.get("votes_cast", {}) if vote_state else {}

    return render_template(
        "now.html",
        song=song,
        contest_active=True,
        votes_remaining=votes_remaining,
        votes_cast=votes_cast,
        total_vote_points=TOTAL_VOTE_POINTS,
        countries=countries,
        error=None,
    )


@app.route("/api/contest/current")
def api_contest_current():
    """JSON proxy for the current contest song (polled by the now-playing page)."""
    data = api_get("/contest/current/")
    if data is None:
        return jsonify({"error": "Backend unavailable"}), 503
    return jsonify(data)


# ---------------------------------------------------------------------------
# Jury routes
# ---------------------------------------------------------------------------


@app.route("/jury")
@login_required("jury")
def jury_page():
    """Jury voting page — point selectors per song."""
    songs_data = api_get("/songs/")
    songs = []
    if songs_data and "payload" in songs_data:
        seen_ids = set()
        for s in songs_data["payload"]:
            if s["songId"] not in seen_ids:
                seen_ids.add(s["songId"])
                songs.append(s)

    # Read the already-cast votes for this jury member from their cookie.
    # Cookie format: { "songId": points, ... }  (both keys and values are ints)
    token = session.get("token", "")
    cookie_name = f"jury_votes_{token}"
    try:
        raw = json.loads(request.cookies.get(cookie_name, "{}"))
        votes_map = (
            {int(k): int(v) for k, v in raw.items()} if isinstance(raw, dict) else {}
        )
    except (ValueError, TypeError):
        votes_map = {}

    used_points = list(votes_map.values())  # point values already awarded
    voted_songs = list(votes_map.keys())  # song IDs already voted on

    return render_template(
        "jury.html",
        songs=songs,
        used_points=used_points,
        voted_songs=voted_songs,
        votes_map=votes_map,
    )


@app.route("/jury/submit", methods=["POST"])
@login_required("jury")
def jury_submit_vote():
    """Handle jury vote submission."""
    token = session.get("token", "")
    song_id = request.form.get("songID")
    points = request.form.get("points")

    if not song_id or not points:
        return jsonify({"error": "Song and points are required"}), 422

    try:
        points_int = int(points)
    except (ValueError, TypeError):
        return jsonify({"error": "Invalid points value"}), 422

    # --- Cookie-based duplicate guard ---
    # Cookie format: { "songId": points, ... }
    cookie_name = f"jury_votes_{token}"
    try:
        raw = json.loads(request.cookies.get(cookie_name, "{}"))
        votes_map = (
            {int(k): int(v) for k, v in raw.items()} if isinstance(raw, dict) else {}
        )
    except (ValueError, TypeError):
        votes_map = {}

    # Reject if this song has already been voted on.
    song_id_int = int(song_id)
    if song_id_int in votes_map:
        return jsonify({"error": "You have already voted for this entry."}), 409

    # Reject if this point value has already been used for another song.
    if points_int in votes_map.values():
        return jsonify(
            {
                "error": f"{points_int} points have already been awarded. Each point value can only be used once."
            }
        ), 409

    status, data, _ = api_post(
        "/jury/vote/",
        params={"Token": token, "songID": song_id, "points": points},
    )

    if status in (200, 202):
        record_vote("jury")
        app.logger.info(
            "jury vote cast",
            extra={"song_id": song_id, "points": points},
        )
        # Persist the updated votes map in the cookie.
        votes_map[song_id_int] = points_int
        response = make_response(jsonify(data), status)
        response.set_cookie(
            cookie_name,
            json.dumps({str(k): v for k, v in votes_map.items()}),
            httponly=True,
            samesite="Strict",
            max_age=60 * 60 * 24 * 7,  # 7 days
        )
        return response

    return jsonify(data), status


# ---------------------------------------------------------------------------
# Error handlers
# ---------------------------------------------------------------------------


@app.errorhandler(422)
def unprocessable_entity(e):
    app.logger.warning("422 unprocessable entity", extra={"path": request.path})
    return render_template(
        "error.html",
        code=422,
        message="Unprocessable Entity — the submitted data was invalid.",
    ), 422


@app.errorhandler(403)
def forbidden(e):
    app.logger.warning("403 forbidden", extra={"path": request.path})
    return render_template(
        "error.html",
        code=403,
        message="Access Denied — you don't have permission to view this page.",
    ), 403


@app.errorhandler(404)
def not_found(e):
    app.logger.info("404 not found", extra={"path": request.path})
    return render_template(
        "error.html",
        code=404,
        message="Page Not Found — this page doesn't exist.",
    ), 404


@app.errorhandler(500)
def internal_error(e):
    app.logger.error(
        "500 internal error", extra={"path": request.path, "error": str(e)}
    )
    return render_template(
        "error.html",
        code=500,
        message="Internal Server Error — something went wrong on our end.",
    ), 500


# ---------------------------------------------------------------------------
# Dev server entry point
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    port = int(os.environ.get("FLASK_PORT", "5000"))
    debug = os.environ.get("FLASK_DEBUG", "false").lower() == "true"
    app.run(host="0.0.0.0", port=port, debug=debug)
