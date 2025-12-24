# Microservice Refactor Notes (ComfyUI Nodes)

## Goal
Outline a refactor direction where node implementations can be deployed as
separate services on Google Cloud, with an emphasis on performance, cost,
and operational simplicity.

## Key Decisions
- Use a control-plane + data-plane split.
- Minimize tensor movement over the network.
- Prefer coarse-grained services over per-node microservices.

## Control Plane (Small Messages)
- Protocol: gRPC with protobufs.
- Payloads: node id, input references, parameters, execution metadata,
  and status/error reporting.
- Deployment: private VPC, internal load balancers, service discovery
  (GKE or equivalent).

## Data Plane (Large Tensors)
- Do not send latents, weights, or large arrays via gRPC.
- Exchange opaque references (URI + shape/dtype metadata).
- Storage options (fastest first):
  - Local NVMe/hostPath on the same node for co-located stages.
  - Filestore (NFS) for low-latency shared storage across pods.
  - Memorystore/Redis for hot intermediates if size allows.
  - GCS for durable artifacts (checkpoints, final images), not per-step
    intermediates.

## Service Granularity
- Avoid a microservice per node if latency-sensitive.
- Group tightly coupled stages (e.g., sampler + VAE decode) into the same
  service/pod to keep tensors local.
- Use separate services only when stages are naturally asynchronous or
  have different scaling/compute profiles.

## Orchestration
- Central orchestrator schedules DAG execution, tracks references, and
  handles retries/timeouts.
- Orchestrator uses gRPC for scheduling and status, and shared storage
  references for data movement.

## Reliability and Scaling
- Idempotent node execution keyed by (node id, inputs, params).
- Cache frequently reused outputs by content hash when possible.
- Backpressure and queueing at the orchestrator, not at individual nodes.

## Security
- Private network for service-to-service traffic.
- Signed URLs or scoped IAM for storage access.
- Audit logging for artifact writes.

## Implementation Notes
- Keep model loading close to compute. Avoid reloading checkpoints per
  request by reusing warm services.
- Explicitly track tensor metadata (shape, dtype, device) with the
  reference for validation and debugging.

## UI
- Keep the current LiteGraph-based UI and interaction model unchanged.
- Continue to render nodes and links with `litegraph.js` and the existing
  ComfyUI frontend layouts, menus, and widgets.
- Replace the backend execution path only: node definitions, workflow
  execution, and status updates should call the orchestrator, which then
  fans out to microservices.
- Preserve the current workflow JSON format and default graph so the UI
  loads and looks identical.
- Maintain existing progress, queue, and error UX; surface status via
  the orchestrator using gRPC (control plane) and a lightweight event
  stream for UI updates.

## Open Questions
- Best shared store for hot tensors in this environment (Filestore vs
  Redis vs local-only with co-location).
- Required latency budget for the default workflow.
- Expected concurrency and autoscaling strategy per service group.

<!-- NODE_MICROSERVICES_START -->
## Node Microservices Inventory

Scope: nodes registered in nodes.py NODE_CLASS_MAPPINGS (core nodes only).

### Orchestrator (control-plane service)
Category: orchestration
Description: Schedules DAG execution, tracks data references, and handles retries/timeouts.
Inputs:
```text
execute_workflow: WorkflowGraphRef
execute_node: { node_id, input_refs, params }
status_request: { workflow_id | node_id }
```
Outputs:
```text
status_events: { state, progress, errors }
output_refs: { node_id, refs }
```

### Load CLIP (`CLIPLoader`)
Category: advanced/loaders
Description: [Recipes]

stable_diffusion: clip-l
stable_cascade: clip-g
sd3: t5 xxl/ clip-g / clip-l
stable_audio: t5 base
mochi: t5 xxl
cosmos: old t5 xxl
lumina2: gemma 2 2B
wan: umt5 xxl
 hidream: llama-3.1 (Recommend) or t5
omnigen2: qwen vl 2.5 3B
Function: load_clip
Inputs:
```text
required:
  clip_name: enum[0]
  type: enum[18]
optional:
  device: enum[2]
```
Outputs:
```text
CLIP
```

### CLIP Set Last Layer (`CLIPSetLastLayer`)
Category: conditioning
Description: Not specified in nodes.py
Function: set_last_layer
Inputs:
```text
required:
  clip: CLIP
  stop_at_clip_layer: INT (default=-1, min=-24, max=-1, step=1)
```
Outputs:
```text
CLIP
```

### CLIP Text Encode (Prompt) (`CLIPTextEncode`)
Category: conditioning
Description: Encodes a text prompt using a CLIP model into an embedding that can be used to guide the diffusion model towards generating specific images.
Function: encode
Inputs:
```text
required:
  text: STRING (multiline=True, dynamicPrompts=True, tooltip=The text to be encoded.)
  clip: CLIP (tooltip=The CLIP model used for encoding the text.)
```
Outputs:
```text
CONDITIONING
```

### CLIP Vision Encode (`CLIPVisionEncode`)
Category: conditioning
Description: Not specified in nodes.py
Function: encode
Inputs:
```text
required:
  clip_vision: CLIP_VISION
  image: IMAGE
  crop: enum[2]
```
Outputs:
```text
CLIP_VISION_OUTPUT
```

### Load CLIP Vision (`CLIPVisionLoader`)
Category: loaders
Description: Not specified in nodes.py
Function: load_clip
Inputs:
```text
required:
  clip_name: enum[0]
```
Outputs:
```text
CLIP_VISION
```

### Load Checkpoint With Config (DEPRECATED) (`CheckpointLoader`)
Category: advanced/loaders
Description: Not specified in nodes.py
Function: load_checkpoint
Inputs:
```text
required:
  config_name: enum[11]
  ckpt_name: enum[0]
```
Outputs:
```text
MODEL, CLIP, VAE
```

### Load Checkpoint (`CheckpointLoaderSimple`)
Category: loaders
Description: Loads a diffusion model checkpoint, diffusion models are used to denoise latents.
Function: load_checkpoint
Inputs:
```text
required:
  ckpt_name: enum[0] (tooltip=The name of the checkpoint (model) to load.)
```
Outputs:
```text
MODEL, CLIP, VAE
```

### ConditioningAverage (`ConditioningAverage`)
Category: conditioning
Description: Not specified in nodes.py
Function: addWeighted
Inputs:
```text
required:
  conditioning_to: CONDITIONING
  conditioning_from: CONDITIONING
  conditioning_to_strength: FLOAT (default=1.0, min=0.0, max=1.0, step=0.01)
```
Outputs:
```text
CONDITIONING
```

### Conditioning (Combine) (`ConditioningCombine`)
Category: conditioning
Description: Not specified in nodes.py
Function: combine
Inputs:
```text
required:
  conditioning_1: CONDITIONING
  conditioning_2: CONDITIONING
```
Outputs:
```text
CONDITIONING
```

### Conditioning (Concat) (`ConditioningConcat`)
Category: conditioning
Description: Not specified in nodes.py
Function: concat
Inputs:
```text
required:
  conditioning_to: CONDITIONING
  conditioning_from: CONDITIONING
```
Outputs:
```text
CONDITIONING
```

### Conditioning (Set Area) (`ConditioningSetArea`)
Category: conditioning
Description: Not specified in nodes.py
Function: append
Inputs:
```text
required:
  conditioning: CONDITIONING
  width: INT (default=64, min=64, max=16384, step=8)
  height: INT (default=64, min=64, max=16384, step=8)
  x: INT (default=0, min=0, max=16384, step=8)
  y: INT (default=0, min=0, max=16384, step=8)
  strength: FLOAT (default=1.0, min=0.0, max=10.0, step=0.01)
```
Outputs:
```text
CONDITIONING
```

### Conditioning (Set Area with Percentage) (`ConditioningSetAreaPercentage`)
Category: conditioning
Description: Not specified in nodes.py
Function: append
Inputs:
```text
required:
  conditioning: CONDITIONING
  width: FLOAT (default=1.0, min=0, max=1.0, step=0.01)
  height: FLOAT (default=1.0, min=0, max=1.0, step=0.01)
  x: FLOAT (default=0, min=0, max=1.0, step=0.01)
  y: FLOAT (default=0, min=0, max=1.0, step=0.01)
  strength: FLOAT (default=1.0, min=0.0, max=10.0, step=0.01)
```
Outputs:
```text
CONDITIONING
```

### ConditioningSetAreaStrength (`ConditioningSetAreaStrength`)
Category: conditioning
Description: Not specified in nodes.py
Function: append
Inputs:
```text
required:
  conditioning: CONDITIONING
  strength: FLOAT (default=1.0, min=0.0, max=10.0, step=0.01)
```
Outputs:
```text
CONDITIONING
```

### Conditioning (Set Mask) (`ConditioningSetMask`)
Category: conditioning
Description: Not specified in nodes.py
Function: append
Inputs:
```text
required:
  conditioning: CONDITIONING
  mask: MASK
  strength: FLOAT (default=1.0, min=0.0, max=10.0, step=0.01)
  set_cond_area: enum[2]
```
Outputs:
```text
CONDITIONING
```

### ConditioningSetTimestepRange (`ConditioningSetTimestepRange`)
Category: advanced/conditioning
Description: Not specified in nodes.py
Function: set_range
Inputs:
```text
required:
  conditioning: CONDITIONING
  start: FLOAT (default=0.0, min=0.0, max=1.0, step=0.001)
  end: FLOAT (default=1.0, min=0.0, max=1.0, step=0.001)
```
Outputs:
```text
CONDITIONING
```

### ConditioningZeroOut (`ConditioningZeroOut`)
Category: advanced/conditioning
Description: Not specified in nodes.py
Function: zero_out
Inputs:
```text
required:
  conditioning: CONDITIONING
```
Outputs:
```text
CONDITIONING
```

### Apply ControlNet (OLD) (`ControlNetApply`)
Category: conditioning/controlnet
Description: Not specified in nodes.py
Function: apply_controlnet
Inputs:
```text
required:
  conditioning: CONDITIONING
  control_net: CONTROL_NET
  image: IMAGE
  strength: FLOAT (default=1.0, min=0.0, max=10.0, step=0.01)
```
Outputs:
```text
CONDITIONING
```

### Apply ControlNet (`ControlNetApplyAdvanced`)
Category: conditioning/controlnet
Description: Not specified in nodes.py
Function: apply_controlnet
Inputs:
```text
required:
  positive: CONDITIONING
  negative: CONDITIONING
  control_net: CONTROL_NET
  image: IMAGE
  strength: FLOAT (default=1.0, min=0.0, max=10.0, step=0.01)
  start_percent: FLOAT (default=0.0, min=0.0, max=1.0, step=0.001)
  end_percent: FLOAT (default=1.0, min=0.0, max=1.0, step=0.001)
optional:
  vae: VAE
```
Outputs:
```text
positive: CONDITIONING, negative: CONDITIONING
```

### Load ControlNet Model (`ControlNetLoader`)
Category: loaders
Description: Not specified in nodes.py
Function: load_controlnet
Inputs:
```text
required:
  control_net_name: enum[0]
```
Outputs:
```text
CONTROL_NET
```

### Load ControlNet Model (diff) (`DiffControlNetLoader`)
Category: loaders
Description: Not specified in nodes.py
Function: load_controlnet
Inputs:
```text
required:
  model: MODEL
  control_net_name: enum[0]
```
Outputs:
```text
CONTROL_NET
```

### DiffusersLoader (`DiffusersLoader`)
Category: advanced/loaders/deprecated
Description: Not specified in nodes.py
Function: load_checkpoint
Inputs:
```text
required:
  model_path: enum[0]
```
Outputs:
```text
MODEL, CLIP, VAE
```

### DualCLIPLoader (`DualCLIPLoader`)
Category: advanced/loaders
Description: [Recipes]

sdxl: clip-l, clip-g
sd3: clip-l, clip-g / clip-l, t5 / clip-g, t5
flux: clip-l, t5
hidream: at least one of t5 or llama, recommended t5 and llama
hunyuan_image: qwen2.5vl 7b and byt5 small
Function: load_clip
Inputs:
```text
required:
  clip_name1: enum[0]
  clip_name2: enum[0]
  type: enum[9]
optional:
  device: enum[2]
```
Outputs:
```text
CLIP
```

### EmptyImage (`EmptyImage`)
Category: image
Description: Not specified in nodes.py
Function: generate
Inputs:
```text
required:
  width: INT (default=512, min=1, max=16384, step=1)
  height: INT (default=512, min=1, max=16384, step=1)
  batch_size: INT (default=1, min=1, max=4096)
  color: INT (default=0, min=0, max=16777215, step=1)
```
Outputs:
```text
IMAGE
```

### Empty Latent Image (`EmptyLatentImage`)
Category: latent
Description: Create a new batch of empty latent images to be denoised via sampling.
Function: generate
Inputs:
```text
required:
  width: INT (default=512, min=16, max=16384, step=8, tooltip=The width of the latent images in pixels.)
  height: INT (default=512, min=16, max=16384, step=8, tooltip=The height of the latent images in pixels.)
  batch_size: INT (default=1, min=1, max=4096, tooltip=The number of latent images in the batch.)
```
Outputs:
```text
LATENT
```

### GLIGENLoader (`GLIGENLoader`)
Category: loaders
Description: Not specified in nodes.py
Function: load_gligen
Inputs:
```text
required:
  gligen_name: enum[0]
```
Outputs:
```text
GLIGEN
```

### GLIGENTextBoxApply (`GLIGENTextBoxApply`)
Category: conditioning/gligen
Description: Not specified in nodes.py
Function: append
Inputs:
```text
required:
  conditioning_to: CONDITIONING
  clip: CLIP
  gligen_textbox_model: GLIGEN
  text: STRING (multiline=True, dynamicPrompts=True)
  width: INT (default=64, min=8, max=16384, step=8)
  height: INT (default=64, min=8, max=16384, step=8)
  x: INT (default=0, min=0, max=16384, step=8)
  y: INT (default=0, min=0, max=16384, step=8)
```
Outputs:
```text
CONDITIONING
```

### Batch Images (`ImageBatch`)
Category: image
Description: Not specified in nodes.py
Function: batch
Inputs:
```text
required:
  image1: IMAGE
  image2: IMAGE
```
Outputs:
```text
IMAGE
```

### Invert Image (`ImageInvert`)
Category: image
Description: Not specified in nodes.py
Function: invert
Inputs:
```text
required:
  image: IMAGE
```
Outputs:
```text
IMAGE
```

### Pad Image for Outpainting (`ImagePadForOutpaint`)
Category: image
Description: Not specified in nodes.py
Function: expand_image
Inputs:
```text
required:
  image: IMAGE
  left: INT (default=0, min=0, max=16384, step=8)
  top: INT (default=0, min=0, max=16384, step=8)
  right: INT (default=0, min=0, max=16384, step=8)
  bottom: INT (default=0, min=0, max=16384, step=8)
  feathering: INT (default=40, min=0, max=16384, step=1)
```
Outputs:
```text
IMAGE, MASK
```

### Upscale Image (`ImageScale`)
Category: image/upscaling
Description: Not specified in nodes.py
Function: upscale
Inputs:
```text
required:
  image: IMAGE
  upscale_method: enum[5]
  width: INT (default=512, min=0, max=16384, step=1)
  height: INT (default=512, min=0, max=16384, step=1)
  crop: enum[2]
```
Outputs:
```text
IMAGE
```

### Upscale Image By (`ImageScaleBy`)
Category: image/upscaling
Description: Not specified in nodes.py
Function: upscale
Inputs:
```text
required:
  image: IMAGE
  upscale_method: enum[5]
  scale_by: FLOAT (default=1.0, min=0.01, max=8.0, step=0.01)
```
Outputs:
```text
IMAGE
```

### InpaintModelConditioning (`InpaintModelConditioning`)
Category: conditioning/inpaint
Description: Not specified in nodes.py
Function: encode
Inputs:
```text
required:
  positive: CONDITIONING
  negative: CONDITIONING
  vae: VAE
  pixels: IMAGE
  mask: MASK
  noise_mask: BOOLEAN (default=True, tooltip=Add a noise mask to the latent so sampling will only happen within the mask. Might improve results or completely break things depending on the model.)
```
Outputs:
```text
positive: CONDITIONING, negative: CONDITIONING, latent: LATENT
```

### KSampler (`KSampler`)
Category: sampling
Description: Uses the provided model, positive and negative conditioning to denoise the latent image.
Function: sample
Inputs:
```text
required:
  model: MODEL (tooltip=The model used for denoising the input latent.)
  seed: INT (default=0, min=0, max=18446744073709551615, tooltip=The random seed used for creating the noise.)
  steps: INT (default=20, min=1, max=10000, tooltip=The number of steps used in the denoising process.)
  cfg: FLOAT (default=8.0, min=0.0, max=100.0, step=0.1, tooltip=The Classifier-Free Guidance scale balances creativity and adherence to the prompt. Higher values result in images more closely matching the prompt however too high values will negatively impact quality.)
  sampler_name: enum[44] (tooltip=The algorithm used when sampling, this can affect the quality, speed, and style of the generated output.)
  scheduler: enum[9] (tooltip=The scheduler controls how noise is gradually removed to form the image.)
  positive: CONDITIONING (tooltip=The conditioning describing the attributes you want to include in the image.)
  negative: CONDITIONING (tooltip=The conditioning describing the attributes you want to exclude from the image.)
  latent_image: LATENT (tooltip=The latent image to denoise.)
  denoise: FLOAT (default=1.0, min=0.0, max=1.0, step=0.01, tooltip=The amount of denoising applied, lower values will maintain the structure of the initial image allowing for image to image sampling.)
```
Outputs:
```text
LATENT
```

### KSampler (Advanced) (`KSamplerAdvanced`)
Category: sampling
Description: Not specified in nodes.py
Function: sample
Inputs:
```text
required:
  model: MODEL
  add_noise: enum[2]
  noise_seed: INT (default=0, min=0, max=18446744073709551615)
  steps: INT (default=20, min=1, max=10000)
  cfg: FLOAT (default=8.0, min=0.0, max=100.0, step=0.1)
  sampler_name: enum[44]
  scheduler: enum[9]
  positive: CONDITIONING
  negative: CONDITIONING
  latent_image: LATENT
  start_at_step: INT (default=0, min=0, max=10000)
  end_at_step: INT (default=10000, min=0, max=10000)
  return_with_leftover_noise: enum[2]
```
Outputs:
```text
LATENT
```

### Latent Blend (`LatentBlend`)
Category: _for_testing
Description: Not specified in nodes.py
Function: blend
Inputs:
```text
required:
  samples1: LATENT
  samples2: LATENT
  blend_factor: FLOAT (default=0.5, min=0, max=1, step=0.01)
```
Outputs:
```text
LATENT
```

### Latent Composite (`LatentComposite`)
Category: latent
Description: Not specified in nodes.py
Function: composite
Inputs:
```text
required:
  samples_to: LATENT
  samples_from: LATENT
  x: INT (default=0, min=0, max=16384, step=8)
  y: INT (default=0, min=0, max=16384, step=8)
  feather: INT (default=0, min=0, max=16384, step=8)
```
Outputs:
```text
LATENT
```

### Crop Latent (`LatentCrop`)
Category: latent/transform
Description: Not specified in nodes.py
Function: crop
Inputs:
```text
required:
  samples: LATENT
  width: INT (default=512, min=64, max=16384, step=8)
  height: INT (default=512, min=64, max=16384, step=8)
  x: INT (default=0, min=0, max=16384, step=8)
  y: INT (default=0, min=0, max=16384, step=8)
```
Outputs:
```text
LATENT
```

### Flip Latent (`LatentFlip`)
Category: latent/transform
Description: Not specified in nodes.py
Function: flip
Inputs:
```text
required:
  samples: LATENT
  flip_method: enum[2]
```
Outputs:
```text
LATENT
```

### Latent From Batch (`LatentFromBatch`)
Category: latent/batch
Description: Not specified in nodes.py
Function: frombatch
Inputs:
```text
required:
  samples: LATENT
  batch_index: INT (default=0, min=0, max=63)
  length: INT (default=1, min=1, max=64)
```
Outputs:
```text
LATENT
```

### Rotate Latent (`LatentRotate`)
Category: latent/transform
Description: Not specified in nodes.py
Function: rotate
Inputs:
```text
required:
  samples: LATENT
  rotation: enum[4]
```
Outputs:
```text
LATENT
```

### Upscale Latent (`LatentUpscale`)
Category: latent
Description: Not specified in nodes.py
Function: upscale
Inputs:
```text
required:
  samples: LATENT
  upscale_method: enum[5]
  width: INT (default=512, min=0, max=16384, step=8)
  height: INT (default=512, min=0, max=16384, step=8)
  crop: enum[2]
```
Outputs:
```text
LATENT
```

### Upscale Latent By (`LatentUpscaleBy`)
Category: latent
Description: Not specified in nodes.py
Function: upscale
Inputs:
```text
required:
  samples: LATENT
  upscale_method: enum[5]
  scale_by: FLOAT (default=1.5, min=0.01, max=8.0, step=0.01)
```
Outputs:
```text
LATENT
```

### Load Image (`LoadImage`)
Category: image
Description: Not specified in nodes.py
Function: load_image
Inputs:
```text
required:
  image: enum[1]
```
Outputs:
```text
IMAGE, MASK
```

### Load Image (as Mask) (`LoadImageMask`)
Category: mask
Description: Not specified in nodes.py
Function: load_image
Inputs:
```text
required:
  image: enum[1]
  channel: enum[4]
```
Outputs:
```text
MASK
```

### Load Image (from Outputs) (`LoadImageOutput`)
Category: image
Description: Load an image from the output folder. When the refresh button is clicked, the node will update the image list and automatically select the first image, allowing for easy iteration.
Function: load_image
Inputs:
```text
required:
  image: COMBO
```
Outputs:
```text
IMAGE, MASK
```

### LoadLatent (`LoadLatent`)
Category: _for_testing
Description: Not specified in nodes.py
Function: load
Inputs:
```text
required:
  latent: list[1]
```
Outputs:
```text
LATENT
```

### Load LoRA (`LoraLoader`)
Category: loaders
Description: LoRAs are used to modify diffusion and CLIP models, altering the way in which latents are denoised such as applying styles. Multiple LoRA nodes can be linked together.
Function: load_lora
Inputs:
```text
required:
  model: MODEL (tooltip=The diffusion model the LoRA will be applied to.)
  clip: CLIP (tooltip=The CLIP model the LoRA will be applied to.)
  lora_name: enum[0] (tooltip=The name of the LoRA.)
  strength_model: FLOAT (default=1.0, min=-100.0, max=100.0, step=0.01, tooltip=How strongly to modify the diffusion model. This value can be negative.)
  strength_clip: FLOAT (default=1.0, min=-100.0, max=100.0, step=0.01, tooltip=How strongly to modify the CLIP model. This value can be negative.)
```
Outputs:
```text
MODEL, CLIP
```

### LoraLoaderModelOnly (`LoraLoaderModelOnly`)
Category: loaders
Description: LoRAs are used to modify diffusion and CLIP models, altering the way in which latents are denoised such as applying styles. Multiple LoRA nodes can be linked together.
Function: load_lora_model_only
Inputs:
```text
required:
  model: MODEL
  lora_name: enum[0]
  strength_model: FLOAT (default=1.0, min=-100.0, max=100.0, step=0.01)
```
Outputs:
```text
MODEL
```

### Preview Image (`PreviewImage`)
Category: image
Description: Saves the input images to your ComfyUI output directory.
Function: save_images
Inputs:
```text
required:
  images: IMAGE
hidden:
  prompt: PROMPT
  extra_pnginfo: EXTRA_PNGINFO
```
Outputs:
```text

```

### Repeat Latent Batch (`RepeatLatentBatch`)
Category: latent/batch
Description: Not specified in nodes.py
Function: repeat
Inputs:
```text
required:
  samples: LATENT
  amount: INT (default=1, min=1, max=64)
```
Outputs:
```text
LATENT
```

### Save Image (`SaveImage`)
Category: image
Description: Saves the input images to your ComfyUI output directory.
Function: save_images
Inputs:
```text
required:
  images: IMAGE (tooltip=The images to save.)
  filename_prefix: STRING (default=ComfyUI, tooltip=The prefix for the file to save. This may include formatting information such as %date:yyyy-MM-dd% or %Empty Latent Image.width% to include values from nodes.)
hidden:
  prompt: PROMPT
  extra_pnginfo: EXTRA_PNGINFO
```
Outputs:
```text

```

### SaveLatent (`SaveLatent`)
Category: _for_testing
Description: Not specified in nodes.py
Function: save
Inputs:
```text
required:
  samples: LATENT
  filename_prefix: STRING (default=latents/ComfyUI)
hidden:
  prompt: PROMPT
  extra_pnginfo: EXTRA_PNGINFO
```
Outputs:
```text

```

### Set Latent Noise Mask (`SetLatentNoiseMask`)
Category: latent/inpaint
Description: Not specified in nodes.py
Function: set_mask
Inputs:
```text
required:
  samples: LATENT
  mask: MASK
```
Outputs:
```text
LATENT
```

### Apply Style Model (`StyleModelApply`)
Category: conditioning/style_model
Description: Not specified in nodes.py
Function: apply_stylemodel
Inputs:
```text
required:
  conditioning: CONDITIONING
  style_model: STYLE_MODEL
  clip_vision_output: CLIP_VISION_OUTPUT
  strength: FLOAT (default=1.0, min=0.0, max=10.0, step=0.001)
  strength_type: enum[2]
```
Outputs:
```text
CONDITIONING
```

### Load Style Model (`StyleModelLoader`)
Category: loaders
Description: Not specified in nodes.py
Function: load_style_model
Inputs:
```text
required:
  style_model_name: enum[0]
```
Outputs:
```text
STYLE_MODEL
```

### Load Diffusion Model (`UNETLoader`)
Category: advanced/loaders
Description: Not specified in nodes.py
Function: load_unet
Inputs:
```text
required:
  unet_name: enum[0]
  weight_dtype: enum[4]
```
Outputs:
```text
MODEL
```

### VAE Decode (`VAEDecode`)
Category: latent
Description: Decodes latent images back into pixel space images.
Function: decode
Inputs:
```text
required:
  samples: LATENT (tooltip=The latent to be decoded.)
  vae: VAE (tooltip=The VAE model used for decoding the latent.)
```
Outputs:
```text
IMAGE
```

### VAE Decode (Tiled) (`VAEDecodeTiled`)
Category: _for_testing
Description: Not specified in nodes.py
Function: decode
Inputs:
```text
required:
  samples: LATENT
  vae: VAE
  tile_size: INT (default=512, min=64, max=4096, step=32)
  overlap: INT (default=64, min=0, max=4096, step=32)
  temporal_size: INT (default=64, min=8, max=4096, step=4, tooltip=Only used for video VAEs: Amount of frames to decode at a time.)
  temporal_overlap: INT (default=8, min=4, max=4096, step=4, tooltip=Only used for video VAEs: Amount of frames to overlap.)
```
Outputs:
```text
IMAGE
```

### VAE Encode (`VAEEncode`)
Category: latent
Description: Not specified in nodes.py
Function: encode
Inputs:
```text
required:
  pixels: IMAGE
  vae: VAE
```
Outputs:
```text
LATENT
```

### VAE Encode (for Inpainting) (`VAEEncodeForInpaint`)
Category: latent/inpaint
Description: Not specified in nodes.py
Function: encode
Inputs:
```text
required:
  pixels: IMAGE
  vae: VAE
  mask: MASK
  grow_mask_by: INT (default=6, min=0, max=64, step=1)
```
Outputs:
```text
LATENT
```

### VAE Encode (Tiled) (`VAEEncodeTiled`)
Category: _for_testing
Description: Not specified in nodes.py
Function: encode
Inputs:
```text
required:
  pixels: IMAGE
  vae: VAE
  tile_size: INT (default=512, min=64, max=4096, step=64)
  overlap: INT (default=64, min=0, max=4096, step=32)
  temporal_size: INT (default=64, min=8, max=4096, step=4, tooltip=Only used for video VAEs: Amount of frames to encode at a time.)
  temporal_overlap: INT (default=8, min=4, max=4096, step=4, tooltip=Only used for video VAEs: Amount of frames to overlap.)
```
Outputs:
```text
LATENT
```

### Load VAE (`VAELoader`)
Category: loaders
Description: Not specified in nodes.py
Function: load_vae
Inputs:
```text
required:
  vae_name: enum[1]
```
Outputs:
```text
VAE
```

### unCLIPCheckpointLoader (`unCLIPCheckpointLoader`)
Category: loaders
Description: Not specified in nodes.py
Function: load_checkpoint
Inputs:
```text
required:
  ckpt_name: enum[0]
```
Outputs:
```text
MODEL, CLIP, VAE, CLIP_VISION
```

### unCLIPConditioning (`unCLIPConditioning`)
Category: conditioning
Description: Not specified in nodes.py
Function: apply_adm
Inputs:
```text
required:
  conditioning: CONDITIONING
  clip_vision_output: CLIP_VISION_OUTPUT
  strength: FLOAT (default=1.0, min=-10.0, max=10.0, step=0.01)
  noise_augmentation: FLOAT (default=0.0, min=0.0, max=1.0, step=0.01)
```
Outputs:
```text
CONDITIONING
```
<!-- NODE_MICROSERVICES_END -->
## gRPC Communication Strategy

- Use protobufs for node inputs/outputs, status, and errors.
- Send only references for large tensors (URI + shape/dtype metadata).
- Keep services on a private VPC with internal load balancing.
- Prefer streaming for status/progress updates.
