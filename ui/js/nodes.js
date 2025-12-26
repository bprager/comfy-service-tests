(() => {
  if (!window.LiteGraph) {
    return;
  }

  const { LiteGraph, LGraphNode } = window;

  function defineNode(type, title, category, configure) {
    function Node() {
      LGraphNode.call(this);
      this.title = title;
      this.serialize_widgets = true;
      configure.call(this);
    }

    Node.title = title;
    Node.category = category;
    Node.prototype = Object.create(LGraphNode.prototype);
    Node.prototype.constructor = Node;

    LiteGraph.registerNodeType(type, Node);
  }

  defineNode(
    "CheckpointLoaderSimple",
    "Checkpoint Loader",
    "loaders",
    function () {
      this.addOutput("MODEL", "MODEL");
      this.addOutput("CLIP", "CLIP");
      this.addOutput("VAE", "VAE");
      this.addWidget(
        "combo",
        "ckpt_name",
        "novaRealityXL_ilV90.safetensors",
        "ckpt_name",
        { values: ["novaRealityXL_ilV90.safetensors"] }
      );
      this.size = [260, 110];
    }
  );

  defineNode("CLIPTextEncode", "CLIP Text Encode", "conditioning", function () {
    this.addInput("clip", "CLIP");
    this.addOutput("CONDITIONING", "CONDITIONING");
    this.addWidget(
      "text",
      "text",
      "a portrait photo, cinematic lighting",
      "text",
      { multiline: true }
    );
    this.size = [320, 160];
  });

  defineNode("EmptyLatentImage", "Empty Latent", "latent", function () {
    this.addOutput("LATENT", "LATENT");
    this.addWidget("number", "width", 512, "width", {
      min: 64,
      max: 2048,
      step: 64,
    });
    this.addWidget("number", "height", 512, "height", {
      min: 64,
      max: 2048,
      step: 64,
    });
    this.addWidget("number", "batch", 1, "batch", {
      min: 1,
      max: 8,
      step: 1,
    });
    this.size = [220, 150];
  });

  defineNode("KSampler", "KSampler", "sampling", function () {
    this.addInput("model", "MODEL");
    this.addInput("positive", "CONDITIONING");
    this.addInput("negative", "CONDITIONING");
    this.addInput("latent_image", "LATENT");
    this.addOutput("LATENT", "LATENT");
    this.addWidget("number", "seed", 0, "seed", {
      min: 0,
      max: 4294967295,
      step: 1,
    });
    this.addWidget("number", "steps", 20, "steps", {
      min: 1,
      max: 100,
      step: 1,
    });
    this.addWidget("number", "cfg", 8, "cfg", {
      min: 1,
      max: 15,
      step: 0.5,
    });
    this.addWidget("combo", "sampler", "euler", "sampler", {
      values: ["euler", "euler_a", "ddim", "heun"],
    });
    this.addWidget("combo", "scheduler", "normal", "scheduler", {
      values: ["normal", "karras", "exponential"],
    });
    this.addWidget("number", "denoise", 1, "denoise", {
      min: 0,
      max: 1,
      step: 0.05,
    });
    this.size = [320, 240];
  });

  defineNode("VAEDecode", "VAE Decode", "latent", function () {
    this.addInput("samples", "LATENT");
    this.addInput("vae", "VAE");
    this.addOutput("IMAGE", "IMAGE");
    this.size = [220, 110];
  });

  defineNode("SaveImage", "Save Image", "image", function () {
    this.addInput("images", "IMAGE");
    this.addWidget("text", "prefix", "ComfyUI", "filename_prefix");
    this.size = [220, 110];
  });

  LiteGraph.auto_sort_node_types = true;
})();
