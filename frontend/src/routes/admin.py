import requests
from flask import Blueprint, current_app, flash, redirect, render_template, request, session, url_for

from api_client import api_delete, api_get, api_post
from config import API_TIMEOUT, EUROSTATS_URL
from utils import login_required

admin_bp = Blueprint("admin", __name__)


@admin_bp.route("/admin")
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


@admin_bp.route("/admin/open", methods=["POST"])
@login_required("admin")
def admin_open_vote():
    """Open voting."""
    token = session.get("token", "")
    status, data, _ = api_post("/admin/open", token=token)
    if status in (200, 202):
        current_app.logger.info("voting opened by admin")
        flash("Voting has been opened!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to open voting.")), "error")
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/admin/close", methods=["POST"])
@login_required("admin")
def admin_close_vote():
    """Close voting."""
    token = session.get("token", "")
    status, data, _ = api_post("/admin/close", token=token)
    if status in (200, 202):
        current_app.logger.info("voting closed by admin")
        flash("Voting has been closed!", "success")
    else:
        flash(
            data.get("message", data.get("error", "Failed to close voting.")), "error"
        )
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/admin/reset", methods=["POST"])
@login_required("admin")
def admin_reset_votes():
    """Reset all votes to zero."""
    token = session.get("token", "")
    status, data = api_delete("/admin/deleteVotes/", token=token)
    if status in (200, 202):
        try:
            requests.post(f"{EUROSTATS_URL}/reset", timeout=API_TIMEOUT)
        except Exception as e:
            current_app.logger.warning(f"Failed to reset EuroStats in-memory state: {e}")
        current_app.logger.warning("all votes reset by admin")
        flash("All votes have been reset!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to reset votes.")), "error")
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/admin/add-country", methods=["POST"])
@login_required("admin")
def admin_add_country():
    """Add a new country."""
    token = session.get("token", "")
    country_id = request.form.get("id", "").strip()
    name = request.form.get("name", "").strip()
    pot = request.form.get("pot", "1").strip()

    if not country_id or not name:
        flash("Country ID and Name are required.", "error")
        return redirect(url_for("admin.admin_dashboard"))

    status, data, _ = api_post(
        "/admin/addCountry/",
        params={"ID": country_id, "Name": name, "Pot": pot},
        token=token,
    )
    if status in (200, 202):
        current_app.logger.info(
            "country added", extra={"country_id": country_id, "country_name": name}
        )
        flash(f"Country '{name}' added successfully!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to add country.")), "error")
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/admin/add-artist", methods=["POST"])
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
        return redirect(url_for("admin.admin_dashboard"))

    status, data, _ = api_post(
        "/admin/addArtist/",
        params={
            "ID": artist_id,
            "Name": last_name,
            "vorName": first_name,
            "typ": artist_type,
            "Land": country,
        },
        token=token,
    )
    if status in (200, 202):
        current_app.logger.info(
            "artist added",
            extra={"artist_id": artist_id, "artist_name": f"{first_name} {last_name}"},
        )
        flash(f"Artist '{first_name} {last_name}' added successfully!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to add artist.")), "error")
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/admin/add-song", methods=["POST"])
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
        return redirect(url_for("admin.admin_dashboard"))

    params = {"ID": artist_id, "Name": song_name, "Land": country}
    if youtube_url:
        params["YoutubeURL"] = youtube_url

    status, data, _ = api_post("/admin/addSong/", params=params, token=token)
    if status in (200, 202):
        current_app.logger.info(
            "song added", extra={"song_name": song_name, "country": country}
        )
        flash(f"Song '{song_name}' added successfully!", "success")
    else:
        flash(data.get("message", data.get("error", "Failed to add song.")), "error")
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/admin/start-contest", methods=["POST"])
@login_required("admin")
def admin_start_contest():
    """Shuffle all songs into a random order and start the contest."""
    token = session.get("token", "")
    status, data, _ = api_post("/admin/startContest", token=token)
    if status in (200, 201):
        song_count = data.get("songCount", 0)
        current_app.logger.info("contest started", extra={"song_count": song_count})
        flash(f"Contest started! {song_count} songs queued in random order.", "success")
    else:
        flash(
            data.get("message", data.get("error", "Failed to start contest.")), "error"
        )
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/admin/advance-contest", methods=["POST"])
@login_required("admin")
def admin_advance_contest():
    """Advance the contest to the next song."""
    token = session.get("token", "")
    status, data, _ = api_post("/admin/advanceContest", token=token)
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
    return redirect(url_for("public.now_playing"))
