FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
  PYTHONUNBUFFERED=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
  git \
  && rm -rf /var/lib/apt/lists/*

COPY services/stage-sampler/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir --upgrade pip \
  && pip install --no-cache-dir torch==2.2.2 --index-url https://download.pytorch.org/whl/cpu \
  && pip install --no-cache-dir -r requirements.txt

COPY proto ./proto
COPY services/stage-sampler/app.py ./app.py
COPY services/stage-sampler/app_core.py ./app_core.py

RUN python -m grpc_tools.protoc \
  -I /app/proto \
  --python_out=/app \
  --grpc_python_out=/app \
  /app/proto/orchestrator.proto

EXPOSE 9091

ENTRYPOINT ["python", "/app/app.py"]
