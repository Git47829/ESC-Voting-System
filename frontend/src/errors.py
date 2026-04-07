from flask import render_template, request


def register_error_handlers(app):
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
