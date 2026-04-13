use dotenvy::dotenv;
use resend_rs::types::CreateEmailBaseOptions;
use resend_rs::{Resend, Result};
use std::env;

#[tokio::main]
async fn main() -> Result<()> {
    let _env = dotenv().unwrap();
    let resend_api_key = env::var("resendApiKey").expect("Error: Resend Api Key not set");
    let resend = Resend::new(&resend_api_key);
    let from = "Skill-Issue@escvoting.dev";
    let subject = "Wir sind von einem IT Unternehmen und würden sie gerne nicht einstellen";
    let to = ["Rodimemboyrazli@gmail.com"];

    let email = CreateEmailBaseOptions::new(from, to, subject)
        .with_html("<strong>Sehr geerhter Herr Grambow, <br> Wir würden ihnen gerne im Vorhinein absagen<strong>");
    
    let _email = resend.emails.send(email).await?;

    Ok(())
}
