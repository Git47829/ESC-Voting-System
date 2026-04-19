import asyncio
import json
import logging
from typing import Callable, Awaitable

import aio_pika

logger = logging.getLogger(__name__)

VOTE_FANOUT_EXCHANGE = "votes.fanout"


class RabbitMQVoteConsumer:
    """Consumes vote messages from RabbitMQ fanout exchange."""

    def __init__(self, rabbitmq_url: str, on_vote: Callable[[dict], Awaitable[None]]):
        self.rabbitmq_url = rabbitmq_url
        self.on_vote = on_vote
        self.connection = None
        self.channel = None

    async def start(self):
        """Connect to RabbitMQ and start consuming votes."""
        while True:
            try:
                await self._connect_and_consume()
            except asyncio.CancelledError:
                logger.info("RabbitMQ consumer cancelled")
                break
            except Exception as e:
                logger.error(f"RabbitMQ consumer error: {e}. Reconnecting in 5s...")
                await asyncio.sleep(5)

    async def _connect_and_consume(self):
        self.connection = await aio_pika.connect_robust(self.rabbitmq_url)
        self.channel = await self.connection.channel()

        # Declare the fanout exchange
        exchange = await self.channel.declare_exchange(
            VOTE_FANOUT_EXCHANGE,
            aio_pika.ExchangeType.FANOUT,
            durable=True,
        )

        # Create exclusive auto-delete queue for this consumer instance
        queue = await self.channel.declare_queue(exclusive=True, auto_delete=True)
        await queue.bind(exchange)

        logger.info(f"Consuming votes from exchange '{VOTE_FANOUT_EXCHANGE}' via queue '{queue.name}'")

        async with queue.iterator() as queue_iter:
            async for message in queue_iter:
                async with message.process():
                    try:
                        vote_data = json.loads(message.body.decode())
                        await self.on_vote(vote_data)
                    except json.JSONDecodeError as e:
                        logger.error(f"Invalid vote message: {e}")
                    except Exception as e:
                        logger.error(f"Error processing vote: {e}")
