use metrics::{counter, histogram};
use resend_rs::types::CreateEmailBaseOptions;
use resend_rs::Resend;
use std::env;
use std::time::Instant;

pub async fn send_mail(recipient: &str, code: &str) -> Result<(), Box<dyn std::error::Error>> {
    let api_key = env::var("RESEND_API_KEY").expect("RESEND_API_KEY not set");
    let resend = Resend::new(&api_key);
    let start = Instant::now();

    let html = format!(
        r#"<!DOCTYPE html>
<html>
<body style="margin:0;padding:0;background:#1a1a2e;font-family:'Segoe UI',Arial,sans-serif;">
  <div style="max-width:420px;margin:40px auto;background:#16213e;border-radius:16px;padding:40px;text-align:center;border:1px solid #0f3460;">
    <h2 style="color:#e2b616;margin:0 0 8px 0;font-size:22px;">ESC Voting System</h2>
    <p style="color:#a0a0b8;margin:0 0 32px 0;font-size:14px;">Dein Bestätigungscode</p>
    <div style="background:#0f3460;border-radius:12px;padding:24px;margin:0 0 24px 0;">
      <p style="font-size:40px;font-weight:bold;letter-spacing:12px;margin:0;color:#e2b616;font-family:'Courier New',monospace;">{}</p>
    </div>
    <p style="color:#a0a0b8;font-size:13px;margin:0;">Dieser Code ist 5 Minuten gültig.</p>
  </div>
</body>
</html>"#,
        code
    );

    let email = CreateEmailBaseOptions::new(
        "ESC Voting <verification@escvoting.dev>",
        [recipient],
        "ESC Voting — Bestätigungscode",
    )
    .with_html(&html);

    let result = resend.emails.send(email).await;
    let duration = start.elapsed();

    histogram!("euromail_email_send_duration_seconds").record(duration);

    match &result {
        Ok(_) => counter!("euromail_emails_sent_total", "status" => "success").increment(1),
        Err(_) => counter!("euromail_emails_sent_total", "status" => "error").increment(1),
    }

    result?;
    Ok(())
}
