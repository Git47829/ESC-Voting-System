import os

API_BASE = os.environ.get("API_BASE_URL", "http://db-crud-api:8000")
API_TIMEOUT = int(os.environ.get("API_TIMEOUT", "10"))
ESC_CONVERTER_URL = os.environ.get(
    "ESC_CONVERTER_URL", "http://public-vote-converter:8090"
)
EUROSTATS_URL = os.environ.get("EUROSTATS_URL", "http://eurostats:8880")

TOTAL_VOTE_POINTS = 20
