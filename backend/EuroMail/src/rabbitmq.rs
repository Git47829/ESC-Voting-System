use lapin::{options::*, types::FieldTable, Connection, ConnectionProperties};
use tracing::{info, error, warn};

use crate::mail;
use crate::models::EmailRequest;

const EMAIL_QUEUE: &str = "email.send";

pub async fn consume_email_queue() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let rabbitmq_url = std::env::var("RABBITMQ_URL")
        .unwrap_or_else(|_| "amqp://guest:guest@rabbitmq:5672/%2f".to_string());

    info!("Connecting to RabbitMQ at {}", rabbitmq_url);

    let conn = Connection::connect(&rabbitmq_url, ConnectionProperties::default()).await?;
    let channel = conn.create_channel().await?;

    channel
        .queue_declare(
            EMAIL_QUEUE,
            QueueDeclareOptions {
                durable: true,
                ..Default::default()
            },
            FieldTable::default(),
        )
        .await?;

    let consumer = channel
        .basic_consume(
            EMAIL_QUEUE,
            "euromail-consumer",
            BasicConsumeOptions::default(),
            FieldTable::default(),
        )
        .await?;

    info!("RabbitMQ consumer started on queue '{}'", EMAIL_QUEUE);

    use futures_lite::StreamExt;
    let mut consumer = consumer;
    while let Some(delivery) = consumer.next().await {
        match delivery {
            Ok(delivery) => {
                match serde_json::from_slice::<EmailRequest>(&delivery.data) {
                    Ok(req) => {
                        info!(recipient = %req.email, "Processing email job from queue");
                        let send_ok = match mail::send_mail(&req.email, &req.token).await {
                            Ok(_) => {
                                info!(recipient = %req.email, "Email sent via queue");
                                true
                            }
                            Err(e) => {
                                error!(recipient = %req.email, error = %e, "Failed to send email from queue");
                                false
                            }
                        };
                        if send_ok {
                            if let Err(e) = delivery.ack(BasicAckOptions::default()).await {
                                error!("Failed to ack message: {}", e);
                            }
                        } else {
                            if let Err(e) = delivery.nack(BasicNackOptions { requeue: true, ..Default::default() }).await {
                                error!("Failed to nack message: {}", e);
                            }
                        }
                    }
                    Err(e) => {
                        warn!("Invalid message payload: {}", e);
                        // Ack invalid messages to avoid infinite retry
                        let _ = delivery.ack(BasicAckOptions::default()).await;
                    }
                }
            }
            Err(e) => {
                error!("Consumer error: {}", e);
            }
        }
    }

    Ok(())
}
