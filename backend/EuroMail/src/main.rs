mod mail;
mod metrics_middleware;
mod models;

use axum::{extract::Json, http::StatusCode, routing::{get, post}, Extension, Router};
use metrics_exporter_prometheus::PrometheusHandle;
use models::EmailRequest;
use tower_http::trace::TraceLayer;
use tracing::{info, error};

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();

    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "euromail=info,tower_http=info".into()),
        )
        .init();

    let metrics_handle = metrics_middleware::setup_metrics();

    let port = std::env::var("PORT").unwrap_or_else(|_| "3000".to_string());
    let app = Router::new()
        .route("/send", post(send_handler))
        .route("/health", get(|| async { StatusCode::OK }))
        .route("/metrics", get(metrics_endpoint))
        .layer(axum::middleware::from_fn(metrics_middleware::track_http_metrics))
        .layer(TraceLayer::new_for_http())
        .layer(Extension(metrics_handle));

    let addr = format!("0.0.0.0:{}", port);
    info!("EuroMail listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .expect("Failed to bind address");

    axum::serve(listener, app).await.expect("Server error");
}

async fn send_handler(Json(req): Json<EmailRequest>) -> StatusCode {
    match mail::send_mail(&req.email, &req.token).await {
        Ok(_) => {
            info!(recipient = %req.email, "Mail sent successfully");
            StatusCode::ACCEPTED
        }
        Err(e) => {
            error!(recipient = %req.email, error = %e, "Failed to send mail");
            StatusCode::INTERNAL_SERVER_ERROR
        }
    }
}

async fn metrics_endpoint(Extension(handle): Extension<PrometheusHandle>) -> String {
    handle.render()
}
