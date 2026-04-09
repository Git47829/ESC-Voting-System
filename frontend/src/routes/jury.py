import json

from flask import (
    Blueprint,
    current_app,
    jsonify,
    make_response,
    render_template,
    request,
    session,
)

from api_client import api_get, api_post
from telemetry import record_vote
from utils import login_required

jury_bp = Blueprint("jury", __name__)


@jury_bp.route("/jury")
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


@jury_bp.route("/jury/submit", methods=["POST"])
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
    cookie_name = f"jury_votes_{token}"
    try:
        raw = json.loads(request.cookies.get(cookie_name, "{}"))
        votes_map = (
            {int(k): int(v) for k, v in raw.items()} if isinstance(raw, dict) else {}
        )
    except (ValueError, TypeError):
        votes_map = {}

    song_id_int = int(song_id)
    if song_id_int in votes_map:
        return jsonify({"error": "You have already voted for this entry."}), 409

    if points_int in votes_map.values():
        return jsonify(
            {
                "error": f"{points_int} points have already been awarded. Each point value can only be used once."
            }
        ), 409

    status, data, _ = api_post(
        "/jury/vote/",
        params={"songID": song_id, "points": points},
        token=token,
    )

    if status in (200, 202):
        record_vote("jury")
        current_app.logger.info(
            "jury vote cast",
            extra={"song_id": song_id, "points": points},
        )
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

    if not isinstance(data, dict):
        data = {"error": "An internal error has occurred while submitting your vote."}

    return jsonify(data), status
