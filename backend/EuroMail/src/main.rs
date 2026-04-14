mod mail;
mod models;

use axum::{extract::Json, http::StatusCode, routing::{get, post}, Router};
use models::EmailRequest;

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();

    let port = std::env::var("PORT").unwrap_or_else(|_| "3000".to_string());
    let app = Router::new()
        .route("/send", post(send_handler))
        .route("/health", get(|| async { StatusCode::OK }));

    let addr = format!("0.0.0.0:{}", port);
    println!("EuroMail listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .expect("Failed to bind address");

    axum::serve(listener, app).await.expect("Server error");
}

async fn send_handler(Json(req): Json<EmailRequest>) -> StatusCode {
    match mail::send_mail(&req.email, &req.token).await {
        Ok(_) => {
            println!("Mail sent to {}", req.email);
            StatusCode::ACCEPTED
        }
        Err(e) => {
            eprintln!("Failed to send mail to {}: {}", req.email, e);
            StatusCode::INTERNAL_SERVER_ERROR
        }
    }
}
