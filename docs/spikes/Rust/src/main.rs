use axum::{Router, routing::get};
use tokio::net::TcpListener;

async fn hello() -> &'static str {
    "Hello, World!"
}

async fn ping() -> &'static str {
    "{\"status\":\"ok\"}"
}

#[tokio::main]
async fn main() {
    let port = std::env::var("PORT").unwrap_or_else(|_| "8080".to_string());
    let app = Router::new()
        .route("/hello", get(hello))
        .route("/ping", get(ping));

    let listener = TcpListener::bind(format!("127.0.0.1:{}", port))
        .await
        .expect("Failed to bind to address");
    axum::serve(listener, app)
        .await
        .expect("Failed to start server");
}
