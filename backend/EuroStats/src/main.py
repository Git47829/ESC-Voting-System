import asyncio
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import JSONResponse
from grpc_consumer import VoteStreamConsumer
from telemetry import setup_telemetry

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

vote_consumer: VoteStreamConsumer = None


async def handle_vote(vote) -> None:
    """
    Handle incoming vote from gRPC stream.
    This is where you can process, store, or broadcast votes.
    """
    logger.info(
        f"Processing vote: song_id={vote.song_id}, "
        f"country={vote.country_voted_for}, "
        f"votes={vote.vote_count}"
    )
    # Hier kannst du deine verarbeitungslogik hinzufügen


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage application lifecycle - startup and shutdown"""
    # Startup
    global vote_consumer
    try:
        vote_consumer = VoteStreamConsumer(
            host="crud-db-api",
            port=50051,
        )
        await vote_consumer.connect()
        logger.info("gRPC consumer initialized")
    except Exception as e:
        logger.error(f"Failed to initialize gRPC consumer: {e}")

    yield

    if vote_consumer:
        await vote_consumer.disconnect()
        logger.info("gRPC consumer disconnected")


app = FastAPI(lifespan=lifespan)

# Initialize OpenTelemetry metrics and tracing
setup_telemetry(app)


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy"}


@app.get("/votes/subscribe")
async def subscribe_to_votes(include_historical: bool = True):
    """
    HTTP endpoint to subscribe to votes stream.
    Returns: JSON-formatted vote data
    """
    if not vote_consumer:
        return JSONResponse(
            status_code=503, content={"error": "gRPC consumer not initialized"}
        )

    votes = []
    try:
        async for vote in vote_consumer.subscribe_to_votes(
            include_historical=include_historical
        ):
            votes.append(
                {
                    "song_id": vote.song_id,
                    "song_name": vote.song_name,
                    "country_voted_for": vote.country_voted_for,
                    "country_voted_for_name": vote.country_voted_for_name,
                    "voter_country": vote.voter_country,
                    "voter_country_name": vote.voter_country_name,
                    "vote_count": vote.vote_count,
                    "timestamp": vote.timestamp,
                }
            )

            if len(votes) >= 100:
                break
    except Exception as e:
        logger.error(f"Error subscribing to votes: {e}")
        return JSONResponse(status_code=500, content={"error": str(e)})

    return {"votes": votes, "count": len(votes)}


#
# Ich habe hier ersteinmal für Testzwecke einen Websocket reingemacht der nur die Daten streamt. Wie man das am ende regelt:
# Rendering am Frontend oder im Backend, überlasse ich dir


@app.websocket("/ws/votes")
async def websocket_votes_endpoint(websocket: WebSocket):
    """
    WebSocket endpoint to stream votes in real-time to connected clients.
    Subscribes to gRPC stream and forwards votes to WebSocket clients.
    """
    await websocket.accept()
    logger.info("WebSocket client connected to votes stream")

    if not vote_consumer:
        await websocket.send_json({"error": "gRPC consumer not initialized"})
        await websocket.close(code=1008)
        return

    try:
        async for vote in vote_consumer.subscribe_to_votes(include_historical=True):
            await websocket.send_json(
                {
                    "type": "vote",
                    "data": {
                        "song_id": vote.song_id,
                        "song_name": vote.song_name,
                        "country_voted_for": vote.country_voted_for,
                        "country_voted_for_name": vote.country_voted_for_name,
                        "voter_country": vote.voter_country,
                        "voter_country_name": vote.voter_country_name,
                        "vote_count": vote.vote_count,
                        "timestamp": vote.timestamp,
                    },
                }
            )

    except WebSocketDisconnect:
        logger.info("WebSocket client disconnected from votes stream")
    except asyncio.CancelledError:
        logger.info("WebSocket connection cancelled")
    except Exception as e:
        logger.error(f"WebSocket error: {e}")
        await websocket.send_json({"type": "error", "message": str(e)})
        await websocket.close(code=1011)
