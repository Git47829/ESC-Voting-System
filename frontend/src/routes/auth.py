from flask import Blueprint, current_app, flash, redirect, render_template, request, session, url_for

from api_client import api_get_auth
from telemetry import set_active_sessions

auth_bp = Blueprint("auth", __name__)


@auth_bp.route("/login", methods=["GET", "POST"])
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
        status, data = api_get_auth("/admin/authenticate", token=token)
        if status != 202:
            current_app.logger.warning("failed admin login attempt")
            flash(data.get("error", "Invalid token."), "error")
            return render_template("login.html"), 401
        session["token"] = token
        session["role"] = "admin"
        set_active_sessions(1)
        current_app.logger.info("admin login", extra={"role": "admin"})
        return redirect(url_for("admin.admin_dashboard"))

    elif role == "jury":
        status, data = api_get_auth("/jury/authenticate", token=token)
        if status != 202:
            current_app.logger.warning("failed jury login attempt")
            flash(data.get("error", "Invalid token."), "error")
            return render_template("login.html"), 401
        session["token"] = token
        session["role"] = "jury"
        set_active_sessions(1)
        current_app.logger.info("jury login", extra={"role": "jury"})
        return redirect(url_for("jury.jury_page"))

    else:
        flash("Invalid role.", "error")
        return render_template("login.html"), 422


@auth_bp.route("/logout")
def logout():
    """Clear session and redirect to home."""
    role = session.get("role", "unknown")
    session.clear()
    set_active_sessions(0)
    current_app.logger.info("logout", extra={"role": role})
    return redirect(url_for("public.vote_page"))
