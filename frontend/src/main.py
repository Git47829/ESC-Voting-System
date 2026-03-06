import os
import time
from functools import wraps

import requests
from flask import (
    Flask,
    abort,
    flash,
    jsonify,
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
        return 500, {"error": str(e)}, {}
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
        return 500, {"error": str(e)}
    finally:
        record_backend_call(endpoint, status_code, time.perf_counter() - t0)


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
    has_voted = request.cookies.get("vote_cookie") is not None
    return render_template("vote.html", songs=songs, has_voted=has_voted)


@app.route("/results")
def results_page():
    """Live results page — bar chart of votes, polled every 10 s."""
    return render_template("results.html")


@app.route("/api/results")
def api_results():
    """JSON endpoint polled by the results page every 10 s."""
    data = api_get("/votes/")
    if data and "payload" in data:
        return jsonify(data["payload"])
    return jsonify([])


@app.route("/vote/submit", methods=["POST"])
def submit_vote():
    """Handle public vote form submission."""
    song_id = request.form.get("songID")
    phone = request.form.get("phoneNum")
    own_country = request.form.get("ownCountry", "")

    if not song_id or not phone:
        return jsonify({"error": "Song and phone number are required"}), 400

    # Forward the browser's vote_cookie to the Go API so it can check it
    browser_cookies = {}
    vote_cookie = request.cookies.get("vote_cookie")
    if vote_cookie:
        browser_cookies["vote_cookie"] = vote_cookie

    status, data, api_cookies = api_post(
        "/vote/",
        params={
            "songID": song_id,
            "phoneNum": phone,
            "ownCountry": own_country,
        },
        cookies=browser_cookies,
    )

    if status == 200:
        record_vote("public")
        app.logger.info(
            "public vote cast",
            extra={"song_id": song_id, "own_country": own_country},
        )

    flask_response = jsonify(data)
    flask_response.status_code = status

    # Forward the vote_cookie from the Go API back to the browser
    if "vote_cookie" in api_cookies:
        flask_response.set_cookie(
            "vote_cookie",
            api_cookies["vote_cookie"],
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
        session["token"] = token
        session["role"] = "admin"
        set_active_sessions(1)
        app.logger.info("admin login", extra={"role": "admin"})
        return redirect(url_for("admin_dashboard"))
    elif role == "jury":
        session["token"] = token
        session["role"] = "jury"
        set_active_sessions(1)
        app.logger.info("jury login", extra={"role": "jury"})
        return redirect(url_for("jury_page"))
    else:
        flash("Invalid role.", "error")
        return render_template("login.html"), 400


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

    if not song_name or not country or not artist_id:
        flash("Song Name, Country, and Artist ID are required.", "error")
        return redirect(url_for("admin_dashboard"))

    status, data, _ = api_post(
        "/admin/addSong/",
        params={"Token": token, "ID": artist_id, "Name": song_name, "Land": country},
    )
    if status in (200, 202):
        app.logger.info(
            "song added", extra={"song_name": song_name, "country": country}
        )
        flash(f"Song '{song_name}' added successfully!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to add song.")), "error")
    return redirect(url_for("admin_dashboard"))


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
    return render_template("jury.html", songs=songs)


@app.route("/jury/submit", methods=["POST"])
@login_required("jury")
def jury_submit_vote():
    """Handle jury vote submission."""
    token = session.get("token", "")
    song_id = request.form.get("songID")
    points = request.form.get("points")

    if not song_id or not points:
        return jsonify({"error": "Song and points are required"}), 400

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

    return jsonify(data), status


# ---------------------------------------------------------------------------
# Error handlers
# ---------------------------------------------------------------------------


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
