# Open questions

- Best shared store for hot tensors in this environment (local, NFS, Redis, or GCS)?
- Required latency budget for the default workflow?
- Expected concurrency and autoscaling strategy per service group?
- Should the UI talk to gRPC via gRPC-web or a lightweight HTTP gateway?
