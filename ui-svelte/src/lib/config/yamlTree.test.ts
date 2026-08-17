import { describe, expect, it } from "vitest";
import {
  addMapEntry,
  addSeqItem,
  buildTree,
  locate,
  parseConfig,
  removeNode,
  renameMapKey,
  serialize,
  setScalarValue,
} from "./yamlTree";

const SAMPLE = `# top comment
logLevel: info
healthCheckTimeout: 120
models:
  demo:
    cmd: |
      echo line1
      echo line2
    name: Demo
    ttl: 0
    caps: [text, image]
peers:
  remote:
    models:
      - a
      - b
`;

describe("yamlTree", () => {
  it("builds a tree with paths and lines", () => {
    const doc = parseConfig(SAMPLE);
    const tree = buildTree(doc);
    const keys = tree.map((n) => n.key);
    expect(keys).toEqual(["logLevel", "healthCheckTimeout", "models", "peers"]);

    const models = tree.find((n) => n.key === "models")!;
    expect(models.kind).toBe("map");
    const demo = models.children.find((c) => c.key === "demo")!;
    const cmd = demo.children.find((c) => c.key === "cmd")!;
    expect(cmd.kind).toBe("scalar");
    expect(cmd.multiline).toBe(true);
    expect(cmd.value).toContain("echo line1");
    expect(cmd.line).toBeGreaterThan(0);
  });

  it("preserves comments and block scalars on round-trip", () => {
    const doc = parseConfig(SAMPLE);
    const out = serialize(doc);
    expect(out).toContain("# top comment");
    expect(out).toContain("echo line1");
    expect(out).toContain("caps: [text, image]");
  });

  it("sets scalar values in place", () => {
    const doc = parseConfig(SAMPLE);
    setScalarValue(doc, ["healthCheckTimeout"], "300");
    expect(locate(doc, ["healthCheckTimeout"])?.node).toMatchObject({ value: 300 });

    setScalarValue(doc, ["logLevel"], "debug");
    expect(locate(doc, ["logLevel"])?.node).toMatchObject({ value: "debug" });

    const out = serialize(doc);
    expect(out).toContain("healthCheckTimeout: 300");
    expect(out).toContain("logLevel: debug");
  });

  it("sets multiline values as block literals", () => {
    const doc = parseConfig(SAMPLE);
    setScalarValue(doc, ["models", "demo", "cmd"], "echo new\n  -m foo");
    const out = serialize(doc);
    expect(out).toContain("echo new");
    expect(out).toContain("-m foo");
  });

  it("adds and removes map entries", () => {
    const doc = parseConfig(SAMPLE);
    const newPath = addMapEntry(doc, ["peers", "remote"]);
    expect(newPath).toEqual(["peers", "remote", ""]);
    renameMapKey(doc, newPath, "apiKey");
    const renamedPath = ["peers", "remote", "apiKey"];
    setScalarValue(doc, renamedPath, "secret");
    expect(locate(doc, renamedPath)?.node).toMatchObject({ value: "secret" });

    removeNode(doc, ["peers", "remote", "apiKey"]);
    expect(locate(doc, ["peers", "remote", "apiKey"])).toBeNull();
  });

  it("adds and removes sequence items", () => {
    const doc = parseConfig(SAMPLE);
    const newPath = addSeqItem(doc, ["peers", "remote", "models"]);
    expect(newPath).toEqual(["peers", "remote", "models", "2"]);
    setScalarValue(doc, newPath, "c");
    expect(locate(doc, ["peers", "remote", "models", "2"])?.node).toMatchObject({ value: "c" });

    removeNode(doc, newPath);
    expect(locate(doc, ["peers", "remote", "models", "2"])).toBeNull();
  });

  it("locates nodes by path", () => {
    const doc = parseConfig(SAMPLE);
    const found = locate(doc, ["models", "demo", "ttl"]);
    expect(found?.node).toMatchObject({ value: 0 });
    expect(locate(doc, ["models", "nope"])).toBeNull();
    expect(locate(doc, ["models", "demo", "cmd", "deep"])).toBeNull();
  });

  it("reports parse errors", () => {
    const doc = parseConfig("models:\n  demo:\n    cmd: [unclosed\n");
    expect(doc.errors.length).toBeGreaterThan(0);
  });
});
