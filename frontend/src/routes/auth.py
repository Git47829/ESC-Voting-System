from flask import Blueprint, current_app, flash, redirect, render_template, request, session, url_for

from api_client import api_post_json
from telemetry import set_active_sessions

auth_bp = Blueprint("auth", __name__)


@auth_bp.route("/login", methods=["GET", "POST"])
def login():
    """Two-step login: credentials then email verification code."""
    if request.method == "GET":
        return render_template("login.html")

    step = request.form.get("step", "1")

    if step == "1":
        email = request.form.get("email", "").strip()
        password = request.form.get("password", "").strip()
        role = request.form.get("role", "admin").strip()

        if not email or not password:
            flash("Email and password are required.", "error")
            return render_template("login.html"), 401

        status, data = api_post_json("/auth/login", {
            "email": email,
            "password": password,
            "role": role,
        })

        if status == 202:
            # Credentials valid, code sent — store temporarily for step 2
            session["pending_email"] = email
            session["pending_password"] = password
            session["pending_role"] = role
            current_app.logger.info("2FA code sent", extra={"email": email, "role": role})
            return render_template("login.html", step=2, email=email, role=role)

        current_app.logger.warning("2FA login failed", extra={"email": email})
        flash(data.get("error", "Invalid credentials."), "error")
        return render_template("login.html"), 401

    elif step == "2":
        email = session.get("pending_email", "")
        password = session.get("pending_password", "")
        role = session.get("pending_role", "")
        code = request.form.get("code", "").strip()

        if not email or not code:
            flash("Verification code is required.", "error")
            return render_template("login.html", step=2, email=email, role=role), 401

        status, data = api_post_json("/auth/verify", {
            "email": email,
            "code": code,
        })

        if status == 202:
            # 2FA verified — create session
            session.pop("pending_email", None)
            session.pop("pending_password", None)
            session.pop("pending_role", None)

            session["token"] = password
            session["email"] = email
            session["role"] = role
            set_active_sessions(1)
            current_app.logger.info("2FA login complete", extra={"email": email, "role": role})

            if role == "admin":
                return redirect(url_for("admin.admin_dashboard"))
            return redirect(url_for("jury.jury_page"))

        current_app.logger.warning("2FA verification failed", extra={"email": email})
        flash(data.get("error", "Invalid or expired code."), "error")
        return render_template("login.html", step=2, email=email, role=role), 401

    flash("Invalid request.", "error")
    return render_template("login.html"), 422


@auth_bp.route("/logout")
def logout():
    """Clear session and redirect to home."""
    role = session.get("role", "unknown")
    session.clear()
    set_active_sessions(0)
    current_app.logger.info("logout", extra={"role": role})
    return redirect(url_for("public.vote_page"))
