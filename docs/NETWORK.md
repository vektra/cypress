How to reliably send a stream over a network
============================================

1. Don't. Just send a stream directly, ie `cypress statsd | nc host:port`.
   This won't ensure that the remote side actually receives messages
   properly, so it's a pretty bad way.

2. Send and ack messages syncronously. For every message transmitted, block
   until the remote side says "ok, I got it.". If the remote side is 20ms away,
   then the max throughput is 25 msg/sec because each send will wait an additional
   20ms to hear the ack before continuing, enlarging the send time to 40ms total.

3. Use a window for verifying reception. If the window is set to 20 messages, then
   those first 20 messages will be transmitted back-to-back without waiting for an ack,
   meaning the remote side might get those 20 messages only 1ms apart, giving the throughput
   of those first messages 1000 msg/sec (subject to TCP windowing too, so it's less).
   When the receiving side gets the first message, it sends an ack back right away, which
   takes 20ms to arrive at the sender. This happens right as the sender has finished sending
   the 20th message and so the window begins to clear on the first 20 messages right
   as the sender wants to transmit more messages. The window then begins to clear at
   the same rate as the sender is sending, meaning the window isn't slowing down the sending,
   allowing the messages to flow at line rate.


Thusly, windowing allows for the fastest sending so long as the window size is tuned
such that `send rate * window size >= transmite time`.
