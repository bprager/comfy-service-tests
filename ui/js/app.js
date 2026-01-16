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
  const inspectorEl = document.getElementById("node-inspector");

  const graph = new LGraph();
  const canvas = new LGraphCanvas("#graph", graph);
  canvas.background_image = "";
  canvas.ds.scale = 1;
  canvas.ds.offset = [0, 0];

  if (
    window.LGraphCanvas &&
    !window.LGraphCanvas.prototype.__comfy_prompt_patched
  ) {
    const originalPrompt = window.LGraphCanvas.prototype.prompt;
    window.LGraphCanvas.prototype.prompt = function (...args) {
      const dialog = originalPrompt ? originalPrompt.apply(this, args) : null;
      if (dialog) {
        const input = dialog.querySelector(".value");
        if (input) {
          input.autocomplete = "off";
          input.autocorrect = "off";
          input.autocapitalize = "off";
          input.spellcheck = false;
          input.name = `lg-prompt-${Date.now()}`;
          input.inputMode = "text";
        }
        dialog.style.zIndex = "1000";
        dialog.style.pointerEvents = "auto";
      }
      return dialog;
    };
    window.LGraphCanvas.prototype.__comfy_prompt_patched = true;
  }

  if (
    window.LGraphCanvas &&
    !window.LGraphCanvas.prototype.__comfy_combo_redraw_patched
  ) {
    const ensureWidgetHitTargets = (node) => {
      if (!node || !Array.isArray(node.widgets) || node.widgets.length === 0) {
        return;
      }
      const lg = window.LiteGraph;
      if (!lg) {
        return;
      }
      let maxY = 0;
      const slotPos = new Float32Array(2);
      if (!node.flags?.collapsed) {
        if (Array.isArray(node.inputs)) {
          node.inputs.forEach((_, index) => {
            const pos = node.getConnectionPos(true, index, slotPos);
            const relativeY = pos[1] - node.pos[1] + lg.NODE_SLOT_HEIGHT * 0.5;
            if (relativeY > maxY) {
              maxY = relativeY;
            }
          });
        }
        if (Array.isArray(node.outputs)) {
          node.outputs.forEach((_, index) => {
            const pos = node.getConnectionPos(false, index, slotPos);
            const relativeY = pos[1] - node.pos[1] + lg.NODE_SLOT_HEIGHT * 0.5;
            if (relativeY > maxY) {
              maxY = relativeY;
            }
          });
        }
      }
      let widgetsY = maxY;
      if (node.horizontal || node.widgets_up) {
        widgetsY = 2;
      }
      if (node.widgets_start_y != null) {
        widgetsY = node.widgets_start_y;
      }
      let posY = widgetsY + 2;
      const width = node.size ? node.size[0] : 0;
      const baseHeight = lg.NODE_WIDGET_HEIGHT || 20;
      node.widgets.forEach((widget) => {
        if (!widget) {
          return;
        }
        const widgetWidth = widget.width || width;
        const y = widget.y ? widget.y : posY;
        widget.last_y = y;
        const size = widget.computeSize ? widget.computeSize(widgetWidth) : null;
        const computed = size ? size[1] : baseHeight;
        const height = Number.isFinite(computed) ? computed : baseHeight;
        posY += height + 4;
      });
    };
    const findWidgetHit = (node, pos) => {
      if (!node || !Array.isArray(node.widgets) || node.widgets.length === 0) {
        return null;
      }
      const lg = window.LiteGraph;
      if (!lg) {
        return null;
      }
      const x = pos[0] - node.pos[0];
      const y = pos[1] - node.pos[1];
      let maxY = 0;
      const slotPos = new Float32Array(2);
      if (!node.flags?.collapsed) {
        if (Array.isArray(node.inputs)) {
          node.inputs.forEach((_, index) => {
            const pos = node.getConnectionPos(true, index, slotPos);
            const relativeY = pos[1] - node.pos[1] + lg.NODE_SLOT_HEIGHT * 0.5;
            if (relativeY > maxY) {
              maxY = relativeY;
            }
          });
        }
        if (Array.isArray(node.outputs)) {
          node.outputs.forEach((_, index) => {
            const pos = node.getConnectionPos(false, index, slotPos);
            const relativeY = pos[1] - node.pos[1] + lg.NODE_SLOT_HEIGHT * 0.5;
            if (relativeY > maxY) {
              maxY = relativeY;
            }
          });
        }
      }
      let widgetsY = maxY;
      if (node.horizontal || node.widgets_up) {
        widgetsY = 2;
      }
      if (node.widgets_start_y != null) {
        widgetsY = node.widgets_start_y;
      }
      let posY = widgetsY + 2;
      const width = node.size ? node.size[0] : 0;
      const baseHeight = lg.NODE_WIDGET_HEIGHT || 20;
      for (const widget of node.widgets) {
        if (!widget) {
          continue;
        }
        const widgetWidth = widget.width || width;
        const widgetY = widget.y ? widget.y : posY;
        const size = widget.computeSize ? widget.computeSize(widgetWidth) : null;
        const computed = size ? size[1] : baseHeight;
        const height = Number.isFinite(computed) ? computed : baseHeight;
        if (!widget.disabled) {
          const withinX = x >= 6 && x <= widgetWidth - 12;
          const withinY = y >= widgetY && y <= widgetY + height;
          if (withinX && withinY) {
            widget.last_y = widgetY;
            return { widget, y: widgetY };
          }
        }
        posY += height + 4;
      }
      return null;
    };
    const originalProcess = window.LGraphCanvas.prototype.processNodeWidgets;
    window.LGraphCanvas.prototype.processNodeWidgets = function (
      node,
      pos,
      event,
      activeWidget
    ) {
      ensureWidgetHitTargets(node);
      const before = Array.isArray(node?.widgets)
        ? node.widgets.map((widget) =>
            widget && widget.type === "combo" ? widget.value : null
          )
        : null;
      const result = originalProcess.call(this, node, pos, event, activeWidget);
      if (
        !result &&
        event &&
        node &&
        Array.isArray(node.widgets) &&
        node.widgets.length
      ) {
        const downEvent = `${window.LiteGraph.pointerevents_method}down`;
        if (event.type === downEvent) {
          const hit = findWidgetHit(node, pos);
          if (hit && hit.widget) {
            return originalProcess.call(this, node, pos, event, hit.widget);
          }
        }
      }
      if (before && Array.isArray(node?.widgets)) {
        const changed = node.widgets.some(
          (widget, index) =>
            widget && widget.type === "combo" && before[index] !== widget.value
        );
        if (changed) {
          this.dirty_area = null;
          this.dirty_canvas = true;
          this.dirty_bgcanvas = true;
        }
      }
      return result;
    };
    window.LGraphCanvas.prototype.__comfy_combo_redraw_patched = true;
  }

  if (
    window.LGraphCanvas &&
    !window.LGraphCanvas.prototype.__comfy_interaction_patched
  ) {
    const originalMouseDown = window.LGraphCanvas.prototype.processMouseDown;
    window.LGraphCanvas.prototype.processMouseDown = function (event) {
      this.allow_interaction = true;
      this.read_only = false;
      this.block_click = false;
      return originalMouseDown.call(this, event);
    };
    const originalMouseUp = window.LGraphCanvas.prototype.processMouseUp;
    window.LGraphCanvas.prototype.processMouseUp = function (event) {
      const result = originalMouseUp.call(this, event);
      this.block_click = false;
      this.pointer_is_down = false;
      return result;
    };
    window.LGraphCanvas.prototype.__comfy_interaction_patched = true;
  }

  LiteGraph.dialog_close_on_mouse_leave = false;
  if (window.LGraphCanvas && window.LGraphCanvas.link_type_colors) {
    window.LGraphCanvas.link_type_colors = {
      ...window.LGraphCanvas.link_type_colors,
      MODEL: "#7aa8ff",
      CLIP: "#f6c453",
      VAE: "#ff8b7b",
      CONDITIONING: "#7fd4a5",
      LATENT: "#e06fc8",
      IMAGE: "#54d0d3",
    };
  }

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
    nodeStates: new Map(),
    lastStatus: "idle",
    pollHandle: null,
    selectedNode: null,
    checkpoints: [],
  };

  function setStatus(text) {
    if (statusEl) {
      statusEl.textContent = text;
    }
    ensureCanvasInteraction();
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

  function ensureCanvasInteraction() {
    if (canvas) {
      canvas.allow_interaction = true;
      canvas.read_only = false;
      canvas.block_click = false;
    }
    if (window.LiteGraph) {
      LiteGraph.dialog_close_on_mouse_leave = false;
    }
  }

  function resetCanvasInteractionState() {
    if (!canvas) {
      return;
    }
    canvas.block_click = false;
    canvas.pointer_is_down = false;
    canvas.last_mouse_dragging = false;
    canvas.dragging_canvas = false;
    canvas.dragging_node = null;
    canvas.connecting_node = null;
    canvas.node_widget = null;
    canvas.node_in_panel = null;
    canvas.last_click_position = null;
    canvas.last_mouseclick = 0;
    ensureCanvasInteraction();
  }

  function forceCanvasRedraw() {
    if (canvas) {
      canvas.dirty_area = null;
      if (typeof canvas.setDirty === "function") {
        canvas.setDirty(true, true);
      }
      if (typeof canvas.draw === "function") {
        canvas.draw(true, true);
        return;
      }
    }
    graph.setDirtyCanvas(true, true);
  }

  function updateWidgetValue(node, widget, rawValue) {
    let value = rawValue;
    if (widget.type === "number") {
      const parsed = Number(rawValue);
      value = Number.isNaN(parsed) ? widget.value : parsed;
    }
    widget.value = value;
    if (widget.options && widget.options.property) {
      if (typeof node.setProperty === "function") {
        node.setProperty(widget.options.property, value);
      } else {
        node.properties = node.properties || {};
        node.properties[widget.options.property] = value;
      }
    }
    if (typeof widget.callback === "function") {
      widget.callback(value, canvas, node, null, null);
    }
    handleGraphEdit();
    forceCanvasRedraw();
  }

  function renderInspector(node) {
    if (!inspectorEl) {
      return;
    }
    inspectorEl.innerHTML = "";
    if (!node || !Array.isArray(node.widgets) || node.widgets.length === 0) {
      const empty = document.createElement("div");
      empty.className = "inspector-empty";
      empty.textContent = "Select a node to edit values.";
      inspectorEl.appendChild(empty);
      return;
    }

    node.widgets.forEach((widget) => {
      if (!widget) {
        return;
      }
      const field = document.createElement("label");
      field.className = "inspector-field";
      const label = document.createElement("span");
      label.textContent = widget.name || "value";
      field.appendChild(label);

      let input;
      const widgetType = widget.type || "";
      if (widgetType === "combo" && widget.options && widget.options.values) {
        input = document.createElement("select");
        widget.options.values.forEach((optionValue) => {
          const option = document.createElement("option");
          option.value = optionValue;
          option.textContent = optionValue;
          input.appendChild(option);
        });
        input.value = widget.value ?? "";
        input.addEventListener("change", () =>
          updateWidgetValue(node, widget, input.value)
        );
      } else if (widgetType === "text" && widget.options?.multiline) {
        input = document.createElement("textarea");
        input.value = widget.value ?? "";
        input.autocomplete = "off";
        input.spellcheck = false;
        input.addEventListener("input", () =>
          updateWidgetValue(node, widget, input.value)
        );
      } else {
        input = document.createElement("input");
        input.type = widgetType === "number" ? "number" : "text";
        if (widgetType === "number" && widget.options) {
          if (widget.options.min !== undefined) {
            input.min = widget.options.min;
          }
          if (widget.options.max !== undefined) {
            input.max = widget.options.max;
          }
          if (widget.options.step !== undefined) {
            input.step = widget.options.step;
          }
        }
        input.value = widget.value ?? "";
        input.autocomplete = "off";
        input.spellcheck = false;
        input.addEventListener("input", () =>
          updateWidgetValue(node, widget, input.value)
        );
      }

      field.appendChild(input);
      inspectorEl.appendChild(field);
    });
  }

  function handleGraphEdit() {
    if (state.lastStatus === "failed") {
      clearNodeStates();
      state.lastStatus = "idle";
      setStatus("Idle");
    }
  }

  function attachNodeEditHandlers() {
    const nodes = graph._nodes || [];
    nodes.forEach((node) => {
      if (!node || node.__comfy_edit_hooked) {
        return;
      }
      const existing = node.onPropertyChanged;
      node.onPropertyChanged = function (...args) {
        if (typeof existing === "function") {
          existing.apply(this, args);
        }
        handleGraphEdit();
        forceCanvasRedraw();
      };
      node.__comfy_edit_hooked = true;
    });
  }

  function applyCheckpointOptions(list) {
    const checkpoints = Array.isArray(list) ? list.filter(Boolean) : [];
    state.checkpoints = checkpoints;
    const nodes = graph._nodes || [];
    nodes.forEach((node) => {
      if (!node || (node.type !== "CheckpointLoaderSimple" && node.type !== "LoadCheckpoint")) {
        return;
      }
      const widget = Array.isArray(node.widgets)
        ? node.widgets.find((item) => item && item.name === "ckpt_name")
        : null;
      if (!widget) {
        return;
      }
      widget.type = "combo";
      widget.options = widget.options || {};
      if (checkpoints.length > 0) {
        widget.options.values = checkpoints;
        if (!checkpoints.includes(widget.value)) {
          updateWidgetValue(node, widget, checkpoints[0]);
        }
      } else {
        widget.options.values = [widget.value || "no checkpoints found"];
      }
    });
    graph.setDirtyCanvas(true, true);
    if (state.selectedNode) {
      renderInspector(state.selectedNode);
    }
  }

  async function loadCheckpoints() {
    try {
      const response = await fetch(`${apiBase}/v1/checkpoints`, {
        cache: "no-store",
      });
      if (!response.ok) {
        throw new Error(`checkpoint list error ${response.status}`);
      }
      const data = await response.json();
      const list = Array.isArray(data.checkpoints) ? data.checkpoints : [];
      applyCheckpointOptions(list);
      appendLog(`Loaded ${list.length} checkpoints.`);
    } catch (err) {
      appendLog(`Checkpoint list error: ${err.message}`);
      applyCheckpointOptions(state.checkpoints);
    }
  }

  function applyNodeState(node, nodeState) {
    if (!node) {
      return;
    }
    if (node.__baseColor === undefined) {
      node.__baseColor = node.color || null;
    }

    if (nodeState === "running") {
      node.color = "#2ecc71";
    } else if (nodeState === "failed") {
      node.color = "#e74c3c";
    } else {
      if (node.__baseColor) {
        node.color = node.__baseColor;
      } else {
        delete node.color;
      }
    }
  }

  function clearNodeStates() {
    state.nodeStates.clear();
    const nodes = graph._nodes || [];
    nodes.forEach((node) => applyNodeState(node, "completed"));
    graph.setDirtyCanvas(true, true);
  }

  function updateNodeStates(nodes) {
    if (!Array.isArray(nodes)) {
      return;
    }
    nodes.forEach((node) => {
      if (!node || node.node_id == null) {
        return;
      }
      const nodeId = Number(node.node_id);
      state.nodeStates.set(nodeId, node.state || "");
      applyNodeState(graph.getNodeById(nodeId), node.state);
    });
    graph.setDirtyCanvas(true, true);
  }

  function setOutput(jobId) {
    if (!outputPreviewEl || !outputMetaEl) {
      return;
    }
    outputPreviewEl.src = `${apiBase}/v1/jobs/${jobId}/output?ts=${Date.now()}`;
    outputMetaEl.textContent = `Job ${jobId} output`;
    ensureCanvasInteraction();
    resetCanvasInteractionState();
  }

  function stopPolling() {
    if (state.pollHandle) {
      clearInterval(state.pollHandle);
      state.pollHandle = null;
    }
  }

  async function pollStatus(jobId) {
    try {
      const response = await fetch(`${apiBase}/v1/jobs/${jobId}`);
      if (!response.ok) {
        return;
      }
      const data = await response.json();
      if (data.status) {
        setStatus(data.status);
        state.lastStatus = data.status;
      }
      if (data.status === "completed") {
        setOutput(jobId);
        resetCanvasInteractionState();
        stopPolling();
      } else if (data.status === "failed") {
        resetCanvasInteractionState();
        stopPolling();
      }
    } catch (err) {
      // Ignore poll failures to avoid noisy UI updates.
    }
  }

  function startPolling(jobId) {
    stopPolling();
    state.pollHandle = setInterval(() => pollStatus(jobId), 3000);
    pollStatus(jobId);
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
      clearNodeStates();
      renderInspector(null);
      attachNodeEditHandlers();
      loadCheckpoints();
      appendLog("Loaded default workflow JSON.");
    } catch (err) {
      appendLog(`Workflow load error: ${err.message}`);
      graph.start();
    }
  }

  async function queueWorkflow() {
    resetCanvasInteractionState();
    clearNodeStates();
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
            state.lastStatus = payload.state;
          }
          if (payload.nodes) {
            updateNodeStates(payload.nodes);
          }
          if (payload.workflow_id && payload.state === "completed") {
            setOutput(payload.workflow_id);
            resetCanvasInteractionState();
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
          if (state.lastStatus === "completed" || state.lastStatus === "failed") {
            setStatus("Idle");
          } else {
            setStatus("Polling for status");
          }
        }
        state.connected = false;
      };
    } catch (err) {
      setStatus("Event stream unavailable");
    }
  }

  async function watchJob(jobId) {
    connectEventStream(jobId);
    startPolling(jobId);
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

  canvas.onNodeSelected = (node) => {
    state.selectedNode = node;
    renderInspector(node);
  };
  canvas.onNodeDeselected = () => {
    state.selectedNode = null;
    renderInspector(null);
  };

  resizeCanvas();
  loadWorkflow();
  if (canvasEl) {
    const unlock = () => ensureCanvasInteraction();
    const reset = () => resetCanvasInteractionState();
    const opts = { capture: true };
    canvasEl.addEventListener("pointerdown", unlock, opts);
    canvasEl.addEventListener("mousedown", unlock, opts);
    canvasEl.addEventListener("touchstart", unlock, opts);
    canvasEl.addEventListener("pointerup", reset, opts);
    canvasEl.addEventListener("mouseup", reset, opts);
    canvasEl.addEventListener("touchend", reset, opts);
    window.addEventListener("pointerup", reset, opts);
    window.addEventListener("mouseup", reset, opts);
    window.addEventListener("touchend", reset, opts);
  }
  // Event stream is opened per job submission.
})();
