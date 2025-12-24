# Constraints

- Go is the primary language for services and shared libraries.
- Control-plane uses gRPC/protobuf for scheduling and status.
- Data-plane moves large tensors via shared storage references only.
- Image manipulation should prefer ImageMagick via the Imagick Go bindings.
- Prefer coarse-grained services that group tightly coupled stages.
- Keep the POC simple: local volumes and docker-compose first.
