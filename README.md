# varz-firehose-nozzle

### Summary
Expose metrics from the Loggregator Firehose to /varz for legacy monitoring applications

### `slowConsumerAlert`
For the most part, the varz-firehose-nozzle emits metrics from the loggregator firehose without processing them too much. A notable exception is the `*.varz.nozzle.slowConsumerAlert` metric. The metric is a binary value (0 or 1) indicating whether or not the nozzle consuming metrics at the same rate that it is receiving them from the firehose: `0` means the the nozzle is keeping up with the firehose, and `1` means that the nozzle is falling behind.

The nozzle determines the value of `slowConsumerAlert` with the following rules:

1. **When the nozzle receives a `TruncatingBuffer.DroppedMessages` metric, it publishes the value `1`.** The metric indicates that Doppler determined that the client (in this case, the nozzle) could not consume messages as quickly as the firehose was sending them, so it dropped messages from its queue of messages to send.

2. **When the nozzle receives a websocket Close frame with status `1011`, it publishes the value `1`.** Traffic Controller pings clients to determine if the connections are still alive. If it does not receive a Pong response before the KeepAlive deadline, it decides that the connection is too slow (or even dead) and sends the Close frame.

3. **Otherwise, the nozzle publishes `0`.**
