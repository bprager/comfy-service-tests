# Open questions

- Best shared store for hot tensors in this environment (local, NFS, Redis, or GCS)?
- Required latency budget for the default workflow?
- Expected concurrency and autoscaling strategy per service group?
- Which checkpoints should be treated as unsupported for CPU-only runs?
