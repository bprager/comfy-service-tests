(() => {
  if (!window.LiteGraph) {
    return;
  }

  const canvasEl = document.getElementById("graph");
  const statusEl = document.getElementById("status-text");
  const queueListEl = document.getElementById("queue-list");
  const logEl = document.getElementById("log-output");
  const gatewayEl = document.getElementById("gateway-target");
  const outputPreviewEl = document.getElementById("output-preview");
  const outputMetaEl = document.getElementById("output-meta");

  const graph = new LGraph();
  const canvas = new LGraphCanvas("#graph", graph);
  canvas.background_image = "";
  canvas.ds.scale = 1;
  canvas.ds.offset = [0, 0];

  const apiBase =
    window.COMFY_API_BASE ||
    `${location.protocol}//${location.hostname || "localhost"}:8084`;
  if (gatewayEl) {
    gatewayEl.textContent = apiBase;
  }

  const state = {
    connected: false,
    jobs: [],
    activeStream: null,
  };

  function setStatus(text) {
    if (statusEl) {
      statusEl.textContent = text;
    }
  }

  function appendLog(message) {
    if (!logEl) {
      return;
    }
    const entry = document.createElement("div");
    entry.textContent = message;
    logEl.appendChild(entry);
    logEl.scrollTop = logEl.scrollHeight;
  }

  function setOutput(jobId) {
    if (!outputPreviewEl || !outputMetaEl) {
      return;
    }
    outputPreviewEl.src = `${apiBase}/v1/jobs/${jobId}/output?ts=${Date.now()}`;
    outputMetaEl.textContent = `Job ${jobId} output`;
  }

  function renderQueue() {
    if (!queueListEl) {
      return;
    }
    queueListEl.innerHTML = "";
    if (!state.jobs.length) {
      const card = document.createElement("div");
      card.className = "queue-item";
      card.textContent = "No jobs queued";
      const meta = document.createElement("span");
      meta.textContent = "Use \"Queue Prompt\" to submit";
      card.appendChild(meta);
      queueListEl.appendChild(card);
      return;
    }
    state.jobs.slice(-3).forEach((job) => {
      const card = document.createElement("div");
      card.className = "queue-item";
      card.textContent = job.title;
      const meta = document.createElement("span");
      meta.textContent = job.status;
      card.appendChild(meta);
      queueListEl.appendChild(card);
    });
  }

  async function loadWorkflow() {
    try {
      const response = await fetch("workflows/default.json", {
        cache: "no-store",
      });
      if (!response.ok) {
        throw new Error("Failed to load default workflow");
      }
      const data = await response.json();
      graph.configure(data);
      graph.start();
      appendLog("Loaded default workflow JSON.");
    } catch (err) {
      appendLog(`Workflow load error: ${err.message}`);
      graph.start();
    }
  }

  async function queueWorkflow() {
    const payload = graph.serialize();
    const job = {
      id: `local-${Date.now()}`,
      title: "Workflow submission",
      status: "queued (local stub)",
    };
    state.jobs.push(job);
    renderQueue();

    try {
      const response = await fetch(`${apiBase}/v1/workflows`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        throw new Error("gateway rejected workflow");
      }
      const result = await response.json();
      if (result.job_id) {
        job.id = result.job_id;
      }
      job.status = `submitted ${result.job_id || ""}`.trim();
      appendLog(`Workflow sent to gateway at ${apiBase}.`);
      if (result.job_id) {
        watchJob(result.job_id);
      }
    } catch (err) {
      job.status = "queued (offline)";
      appendLog(`Gateway error: ${err.message}`);
    }
    renderQueue();
  }

  function connectEventStream(jobId) {
    if (state.activeStream) {
      state.activeStream.close();
      state.activeStream = null;
    }
    const eventsUrl = jobId
      ? `${apiBase}/v1/events?id=${jobId}`
      : `${apiBase}/v1/events`;
    if (!window.EventSource) {
      setStatus("Event stream unsupported");
      return;
    }
    try {
      const source = new EventSource(eventsUrl);
      state.activeStream = source;
      source.onopen = () => {
        state.connected = true;
        setStatus(`Connected to ${eventsUrl}`);
        appendLog("Event stream connected.");
      };
      source.onmessage = (event) => {
        appendLog(`Event: ${event.data}`);
        try {
          const payload = JSON.parse(event.data);
          if (payload.state) {
            setStatus(payload.state);
          }
          if (payload.workflow_id && payload.state === "completed") {
            setOutput(payload.workflow_id);
          }
          if (payload.workflow_id) {
            const match = state.jobs.find((item) => item.id === payload.workflow_id);
            if (match) {
              match.status = payload.state || match.status;
              renderQueue();
            }
          }
        } catch (err) {
          // Ignore malformed payloads.
        }
      };
      source.onerror = () => {
        if (state.connected) {
          setStatus("Event stream disconnected");
        }
        state.connected = false;
      };
    } catch (err) {
      setStatus("Event stream unavailable");
    }
  }

  async function watchJob(jobId) {
    connectEventStream(jobId);
    try {
      const response = await fetch(`${apiBase}/v1/jobs/${jobId}`);
      if (response.ok) {
        const data = await response.json();
        setStatus(data.status || "queued");
        if (data.status === "completed") {
          setOutput(jobId);
        }
      }
    } catch (err) {
      appendLog(`Status check failed: ${err.message}`);
    }
  }

  function resizeCanvas() {
    const rect = canvasEl.getBoundingClientRect();
    canvas.resize(rect.width, rect.height);
  }

  window.addEventListener("resize", resizeCanvas);

  const queueButton = document.getElementById("queue-btn");
  if (queueButton) {
    queueButton.addEventListener("click", queueWorkflow);
  }

  const resetButton = document.getElementById("reset-btn");
  if (resetButton) {
    resetButton.addEventListener("click", loadWorkflow);
  }

  resizeCanvas();
  loadWorkflow();
  // Event stream is opened per job submission.
})();
