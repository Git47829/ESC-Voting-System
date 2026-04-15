use axum::{extract::Request, middleware::Next, response::Response};
use metrics::{counter, describe_counter, describe_histogram, histogram, Unit};
use metrics_exporter_prometheus::{Matcher, PrometheusBuilder, PrometheusHandle};
use std::time::Instant;

pub fn setup_metrics() -> PrometheusHandle {
    describe_counter!(
        "euromail_http_requests_total",
        Unit::Count,
        "Total HTTP requests received"
    );
    describe_histogram!(
        "euromail_http_request_duration_seconds",
        Unit::Seconds,
        "HTTP request latency distribution"
    );
    describe_counter!(
        "euromail_emails_sent_total",
        Unit::Count,
        "Total emails sent"
    );
    describe_histogram!(
        "euromail_email_send_duration_seconds",
        Unit::Seconds,
        "Email send latency distribution"
    );

    PrometheusBuilder::new()
        .set_buckets_for_metric(
            Matcher::Prefix("euromail_http_request_duration".to_string()),
            &[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5],
        )
        .unwrap()
        .set_buckets_for_metric(
            Matcher::Prefix("euromail_email_send_duration".to_string()),
            &[0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0],
        )
        .unwrap()
        .install_recorder()
        .expect("failed to install Prometheus recorder")
}

pub async fn track_http_metrics(req: Request, next: Next) -> Response {
    let method = req.method().to_string();
    let path = req.uri().path().to_string();
    let start = Instant::now();

    let response = next.run(req).await;

    let status = response.status().as_u16().to_string();
    let duration = start.elapsed();

    counter!(
        "euromail_http_requests_total",
        "method" => method.clone(),
        "path" => path.clone(),
        "status" => status,
    )
    .increment(1);

    histogram!(
        "euromail_http_request_duration_seconds",
        "method" => method,
        "path" => path,
    )
    .record(duration);

    response
}
