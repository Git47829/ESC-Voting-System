import time

import requests
from flask import Blueprint, current_app, jsonify, make_response, render_template, request

from api_client import api_get, api_post
from config import API_TIMEOUT, ESC_CONVERTER_URL, TOTAL_VOTE_POINTS
from telemetry import record_backend_call, record_vote
from utils import decode_vote_state_cookie, normalize_youtube_url

public_bp = Blueprint("public", __name__)


@public_bp.route("/")
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


@public_bp.route("/results")
def results_page():
    """Live results page — bar chart of votes, polled every 10 s."""
    return render_template("results.html")


@public_bp.route("/stats")
def stats_page():
    """Live statistics page — pie charts from EuroStats WebSocket."""
    return render_template("stats.html")


@public_bp.route("/api/results")
def api_results():
    """Combined jury + ESC-converted public ranking, computed on-the-fly."""
    t0 = time.perf_counter()
    esc_status = 500
    try:
        esc_resp = requests.get(
            f"{ESC_CONVERTER_URL}/api/esc-points", timeout=API_TIMEOUT
        )
        esc_status = esc_resp.status_code
        esc_resp.raise_for_status()
        esc_data = esc_resp.json()
    except requests.exceptions.RequestException as e:
        current_app.logger.error("converter GET failed", extra={"error": str(e)})
        return jsonify([]), 503
    finally:
        record_backend_call("/api/esc-points", esc_status, time.perf_counter() - t0)

    jury_data = api_get("/votes/")

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


@public_bp.route("/vote/submit", methods=["POST"])
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
        current_app.logger.info(
            "public vote cast",
            extra={"song_id": song_id, "own_country": own_country, "points": points},
        )

    flask_response = make_response(jsonify(data), status)

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


@public_bp.route("/now")
def now_playing():
    """Running Now page — shows the current song with YouTube embed and voting."""
    data = api_get("/contest/current")
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


@public_bp.route("/api/contest/current")
def api_contest_current():
    """JSON proxy for the current contest song (polled by the now-playing page)."""
    data = api_get("/contest/current/")
    if data is None:
        return jsonify({"error": "Backend unavailable"}), 503
    return jsonify(data)
