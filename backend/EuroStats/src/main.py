import asyncio
import base64
import io
import logging
import os
from contextlib import asynccontextmanager
from typing import List

import matplotlib
matplotlib.use("Agg")  # must be called before any other matplotlib import
import matplotlib.patches as mpatches
import matplotlib.pyplot as plt
import pandas as pd
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import JSONResponse
from grpc_consumer import VoteStreamConsumer
from telemetry import setup_telemetry

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

vote_consumer: VoteStreamConsumer = None

# ---------------------------------------------------------------------------
# In-memory vote store
# ---------------------------------------------------------------------------
_vote_df: pd.DataFrame = pd.DataFrame(columns=[
    "song_id", "song_name",
    "country_voted_for", "country_voted_for_name",
    "voter_country", "voter_country_name",
    "vote_count", "timestamp",
])

# ---------------------------------------------------------------------------
# WebSocket connection manager for /ws/stats
# ---------------------------------------------------------------------------
class StatsConnectionManager:
    def __init__(self):
        self.active_connections: List[WebSocket] = []

    async def connect(self, websocket: WebSocket):
        await websocket.accept()
        self.active_connections.append(websocket)

    def disconnect(self, websocket: WebSocket):
        if websocket in self.active_connections:
            self.active_connections.remove(websocket)

    async def broadcast(self, message: dict):
        dead = []
        for conn in self.active_connections:
            try:
                await conn.send_json(message)
            except Exception:
                dead.append(conn)
        for d in dead:
            self.active_connections.remove(d)


stats_manager = StatsConnectionManager()

# ---------------------------------------------------------------------------
# Chart colour palette (ESC dark theme)
# ---------------------------------------------------------------------------
_BG_COLOR = "#0a0a0a"
_TEXT_COLOR = "#e8e8e8"
_YELLOW = "#ffde00"

_SLICE_COLORS = [
    "#ffde00", "#e05c5c", "#5c9de0", "#5ce09d", "#e09d5c",
    "#9d5ce0", "#5ce0e0", "#e05ce0", "#9de05c", "#e0c65c",
    "#5c6de0", "#e07d5c", "#5ce07d", "#c6e05c", "#7d5ce0",
    "#5cc6e0", "#e05c9d", "#9de0c6", "#c65ce0", "#e0e05c",
]


def _make_pie_chart(labels: list, values: list, title: str) -> str:
    """Render a 100% pie chart and return it as a base64-encoded PNG data URL."""
    fig, ax = plt.subplots(figsize=(7, 7), facecolor=_BG_COLOR)
    ax.set_facecolor(_BG_COLOR)

    n = len(labels)
    colors = (_SLICE_COLORS * ((n // len(_SLICE_COLORS)) + 1))[:n]

    _, _, autotexts = ax.pie(
        values,
        labels=None,
        colors=colors,
        autopct="%1.1f%%",
        pctdistance=0.78,
        startangle=90,
        wedgeprops={"linewidth": 0.5, "edgecolor": _BG_COLOR},
    )

    for autotext in autotexts:
        autotext.set_color(_TEXT_COLOR)
        autotext.set_fontsize(8)

    legend_patches = [
        mpatches.Patch(color=colors[i], label=labels[i])
        for i in range(n)
    ]
    ax.legend(
        handles=legend_patches,
        loc="lower center",
        bbox_to_anchor=(0.5, -0.08),
        ncol=3,
        fontsize=8,
        framealpha=0,
        labelcolor=_TEXT_COLOR,
    )

    ax.set_title(title, color=_YELLOW, fontsize=13, fontweight="bold", pad=14)

    buf = io.BytesIO()
    fig.savefig(buf, format="png", bbox_inches="tight", facecolor=_BG_COLOR, dpi=120)
    plt.close(fig)
    buf.seek(0)
    encoded = base64.b64encode(buf.read()).decode("utf-8")
    return f"data:image/png;base64,{encoded}"


# ---------------------------------------------------------------------------
# Stats computation and broadcast
# ---------------------------------------------------------------------------
async def _compute_and_broadcast() -> None:
    """Recompute both pie charts and broadcast to all /ws/stats clients."""
    if not stats_manager.active_connections:
        return

    if _vote_df.empty:
        await stats_manager.broadcast({"type": "stats", "charts": None, "vote_count": 0})
        return

    voter_counts = (
        _vote_df.groupby("voter_country_name")["vote_count"]
        .sum()
        .sort_values(ascending=False)
    )
    chart1 = _make_pie_chart(
        labels=voter_counts.index.tolist(),
        values=voter_counts.values.tolist(),
        title="Voters by Country",
    )

    received_counts = (
        _vote_df.groupby("country_voted_for_name")["vote_count"]
        .sum()
        .sort_values(ascending=False)
    )
    chart2 = _make_pie_chart(
        labels=received_counts.index.tolist(),
        values=received_counts.values.tolist(),
        title="Votes Received by Country",
    )

    await stats_manager.broadcast({
        "type": "stats",
        "vote_count": int(_vote_df["vote_count"].sum()),
        "charts": {
            "voters_by_country": chart1,
            "votes_received_by_country": chart2,
        },
    })


# ---------------------------------------------------------------------------
# Vote handler — appends to DataFrame and triggers broadcast
# ---------------------------------------------------------------------------
async def handle_vote(vote) -> None:
    global _vote_df
    logger.info(
        f"Processing vote: song_id={vote.song_id}, "
        f"country={vote.country_voted_for}, "
        f"votes={vote.vote_count}"
    )
    new_row = pd.DataFrame([{
        "song_id": vote.song_id,
        "song_name": vote.song_name,
        "country_voted_for": vote.country_voted_for,
        "country_voted_for_name": vote.country_voted_for_name,
        "voter_country": vote.voter_country,
        "voter_country_name": vote.voter_country_name,
        "vote_count": vote.vote_count,
        "timestamp": vote.timestamp,
    }])
    _vote_df = pd.concat([_vote_df, new_row], ignore_index=True)
    await _compute_and_broadcast()


# ---------------------------------------------------------------------------
# Background task: single gRPC stream consumer for state accumulation
# ---------------------------------------------------------------------------
async def _run_vote_ingestor():
    while True:
        try:
            async for vote in vote_consumer.subscribe_to_votes(include_historical=True):
                await handle_vote(vote)
        except Exception as e:
            logger.error(f"Vote ingestor error, retrying in 5s: {e}")
            await asyncio.sleep(5)

    # Store timestamp as int (seconds) to keep DataFrame rows JSON-serializable
    ts = int(vote.timestamp.seconds) if hasattr(vote.timestamp, "seconds") else int(vote.timestamp)

    new_row = pd.DataFrame([{
        "song_id": int(vote.song_id),
        "song_name": vote.song_name,
        "country_voted_for": vote.country_voted_for,
        "country_voted_for_name": vote.country_voted_for_name,
        "voter_country": vote.voter_country,
        "voter_country_name": vote.voter_country_name,
        "vote_count": int(vote.vote_count),
        "timestamp": ts,
    }])
    _vote_df = pd.concat([_vote_df, new_row], ignore_index=True)

# ---------------------------------------------------------------------------
# Application lifecycle
# ---------------------------------------------------------------------------
@asynccontextmanager
async def lifespan(app: FastAPI):
    global vote_consumer
    try:
        grpc_host = os.getenv("GRPC_HOST", "db-crud-api")
        grpc_port = int(os.getenv("GRPC_PORT", "50051"))
        vote_consumer = VoteStreamConsumer(host=grpc_host, port=grpc_port)
        await vote_consumer.connect()
        logger.info("gRPC consumer initialized")
        asyncio.create_task(_run_vote_ingestor())
        logger.info("Vote ingestor background task started")
    except Exception as e:
        logger.error(f"Failed to initialize gRPC consumer: {e}")

    yield

    if vote_consumer:
        await vote_consumer.disconnect()
        logger.info("gRPC consumer disconnected")


app = FastAPI(lifespan=lifespan)

# Initialize OpenTelemetry metrics and tracing
setup_telemetry(app)


# ---------------------------------------------------------------------------
# HTTP endpoints
# ---------------------------------------------------------------------------
@app.get("/health")
async def health_check():
    return {"status": "healthy"}


@app.get("/votes/subscribe")
async def subscribe_to_votes(include_historical: bool = True):
    """HTTP endpoint to subscribe to votes stream. Returns up to 100 votes."""
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


# ---------------------------------------------------------------------------
# WebSocket: raw vote stream
# ---------------------------------------------------------------------------
@app.websocket("/ws/votes")
async def websocket_votes_endpoint(websocket: WebSocket):
    """Stream raw vote events to connected clients."""
    await websocket.accept()
    logger.info("WebSocket client connected to votes stream")
    try:
        # Send snapshot of all accumulated votes so late-joiners get current state
        if not _vote_df.empty:
            snapshot = _vote_df.to_dict(orient="records")
            await websocket.send_json({"type": "snapshot", "data": snapshot})
        # Keep alive with periodic pings
        while True:
            await asyncio.sleep(30)
            await websocket.send_json({"type": "ping"})
    except WebSocketDisconnect:
        logger.info("WebSocket client disconnected from votes stream")
    except asyncio.CancelledError:
        logger.info("WebSocket votes connection cancelled")
    except Exception as e:
        logger.error(f"Votes WebSocket error: {e}")
    finally:
        votes_manager.disconnect(websocket)


# ---------------------------------------------------------------------------
# WebSocket: live statistics charts
# ---------------------------------------------------------------------------
@app.websocket("/ws/stats")
async def websocket_stats_endpoint(websocket: WebSocket):
    """Push matplotlib pie charts to connected clients whenever votes change."""
    await stats_manager.connect(websocket)
    logger.info("WebSocket client connected to stats stream")
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
        logger.info("WebSocket stats client disconnected")
    except asyncio.CancelledError:
        logger.info("WebSocket stats connection cancelled")
    except Exception as e:
        logger.error(f"WebSocket error: {e}")
        await websocket.send_json({"type": "error", "message": str(e)})
        await websocket.close(code=1011)


# ---------------------------------------------------------------------------
# WebSocket: live statistics charts
# ---------------------------------------------------------------------------
@app.websocket("/ws/stats")
async def websocket_stats_endpoint(websocket: WebSocket):
    """Push matplotlib pie charts to connected clients whenever votes change."""
    await stats_manager.connect(websocket)
    logger.info("WebSocket client connected to stats stream")
    try:
        # Send current state immediately on connect
        await _compute_and_broadcast()
        # Keep alive with periodic pings
        while True:
            await asyncio.sleep(30)
            await websocket.send_json({"type": "ping"})
    except WebSocketDisconnect:
        logger.info("WebSocket stats client disconnected")
    except asyncio.CancelledError:
        logger.info("WebSocket stats connection cancelled")
    except Exception as e:
        logger.error(f"Stats WebSocket error: {e}")
    finally:
        stats_manager.disconnect(websocket)
