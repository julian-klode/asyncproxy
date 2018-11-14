# asyncproxy - Improve initial connection speed

asyncproxy is a http proxy that keeps connections in the background
for CONNECT requests done previously, so if a new CONNECT request
comes in for the same host, it can just use the established connection
and establish a new one for later use in the background.

This reduces the RTT for CONNECT requests on slow connections.
