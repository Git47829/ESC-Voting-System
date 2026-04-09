import os

from flask import Flask

app = Flask(__name__)
app.secret_key = os.environ.get("FLASK_SECRET_KEY", "esc-voting-secret-key-change-me")

from telemetry import setup_telemetry  # noqa: E402 — intentional post-app import

setup_telemetry(app)

from routes.admin import admin_bp
from routes.auth import auth_bp
from routes.jury import jury_bp
from routes.public import public_bp

app.register_blueprint(public_bp)
app.register_blueprint(auth_bp)
app.register_blueprint(admin_bp)
app.register_blueprint(jury_bp)

from errors import register_error_handlers

register_error_handlers(app)

from utils import get_voting_status


@app.context_processor
def inject_voting_status():
    try:
        is_open = get_voting_status()
    except Exception:
        is_open = False
    return dict(voting_is_open=is_open)


if __name__ == "__main__":
    port = int(os.environ.get("FLASK_PORT", "5000"))
    debug = os.environ.get("FLASK_DEBUG", "false").lower() == "true"
    app.run(host="0.0.0.0", port=port, debug=debug)
