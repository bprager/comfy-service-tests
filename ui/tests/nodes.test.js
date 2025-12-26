const test = require("node:test");
const assert = require("node:assert/strict");

function FakeGraphNode() {
  this.widgets = [];
}

FakeGraphNode.prototype.addInput = function () {};
FakeGraphNode.prototype.addOutput = function () {};
FakeGraphNode.prototype.addWidget = function (
  type,
  name,
  value,
  property,
  options = {}
) {
  const widget = { type, name, value, property, options };
  this.widgets.push(widget);
  return widget;
};

function loadNodes(registry) {
  delete require.cache[require.resolve("../js/nodes.js")];
  global.window = {
    LiteGraph: {
      registerNodeType(type, ctor) {
        registry[type] = ctor;
      },
      auto_sort_node_types: false,
    },
    LGraphNode: FakeGraphNode,
  };
  require("../js/nodes.js");
}

test("CLIPTextEncode uses multiline text widget", () => {
  const registry = {};
  loadNodes(registry);

  assert.ok(registry.CLIPTextEncode, "CLIPTextEncode should be registered");
  const node = new registry.CLIPTextEncode();
  const widget = node.widgets.find((item) => item.name === "text");
  assert.ok(widget, "text widget should exist");
  assert.equal(widget.type, "text");
  assert.equal(widget.options.multiline, true);
});

test("SaveImage exposes prefix text widget", () => {
  const registry = {};
  loadNodes(registry);

  assert.ok(registry.SaveImage, "SaveImage should be registered");
  const node = new registry.SaveImage();
  const widget = node.widgets.find((item) => item.name === "prefix");
  assert.ok(widget, "prefix widget should exist");
  assert.equal(widget.type, "text");
  assert.equal(widget.property, "filename_prefix");
});

test("Checkpoint loader uses combo widget", () => {
  const registry = {};
  loadNodes(registry);

  assert.ok(registry.CheckpointLoaderSimple, "Checkpoint loader should be registered");
  const node = new registry.CheckpointLoaderSimple();
  const widget = node.widgets.find((item) => item.name === "ckpt_name");
  assert.ok(widget, "ckpt_name widget should exist");
  assert.equal(widget.type, "combo");
  assert.deepEqual(widget.options.values, ["novaRealityXL_ilV90.safetensors"]);
});

test("Custom nodes serialize widget values", () => {
  const registry = {};
  loadNodes(registry);

  const node = new registry.CheckpointLoaderSimple();
  assert.equal(node.serialize_widgets, true);
});
