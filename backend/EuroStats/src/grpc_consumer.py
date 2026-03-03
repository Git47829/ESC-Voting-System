import asyncio
import logging
from typing import AsyncGenerator

import grpc
from proto import votes_pb2, votes_pb2_grpc

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class VoteStreamConsumer:
    """Consumes vote data from the gRPC VoteService"""

    def __init__(self, host: str = "localhost", port: int = 50051):
        """
        Initialize the VoteStreamConsumer.

        Args:
            host: The gRPC server host
            port: The gRPC server port
        """
        self.host = host
        self.port = port
        self.channel = None
        self.stub = None

    async def connect(self) -> None:
        """Establish connection to the gRPC server"""
        try:
            self.channel = grpc.aio.secure_channel(
                f"{self.host}:{self.port}",
                grpc.ssl_channel_credentials(),
            )
            self.stub = votes_pb2_grpc.VoteServiceStub(self.channel)
            logger.info(f"Connected to gRPC server at {self.host}:{self.port}")
        except Exception as e:
            logger.error(f"Failed to connect to gRPC server: {e}")
            raise

    async def disconnect(self) -> None:
        """Close the connection to the gRPC server"""
        if self.channel:
            await self.channel.close()
            logger.info("Disconnected from gRPC server")

    async def subscribe_to_votes(
        self, include_historical: bool = True
    ) -> AsyncGenerator[votes_pb2.Vote, None]:
        """
        Subscribe to the vote stream from the gRPC server.

        Args:
            include_historical: Whether to include historical votes

        Yields:
            Vote messages from the stream
        """
        try:
            request = votes_pb2.VoteStreamRequest(include_historical=include_historical)
            logger.info(
                f"Subscribing to votes stream (historical={include_historical})"
            )

            async for vote in self.stub.StreamVotes(request):
                logger.debug(
                    f"Received vote: song_id={vote.song_id}, "
                    f"country={vote.country_voted_for}, "
                    f"votes={vote.vote_count}"
                )
                yield vote

        except grpc.aio.AioRpcError as e:
            logger.error(f"gRPC error: {e.code()}: {e.details()}")
            raise
        except Exception as e:
            logger.error(f"Error subscribing to votes: {e}")
            raise

    async def process_votes(self, process_fn, include_historical: bool = True) -> None:
        """
        Process votes as they arrive from the stream.

        Args:
            process_fn: Async function to process each vote
            include_historical: Whether to include historical votes
        """
        try:
            async for vote in self.subscribe_to_votes(
                include_historical=include_historical
            ):
                await process_fn(vote)
        except asyncio.CancelledError:
            logger.info("Vote processing cancelled")
        except Exception as e:
            logger.error(f"Error processing votes: {e}")
            raise
