use serde::{Deserialize, Serialize};

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct EmailRequest {
    pub email: String,
    pub token: String,
}
