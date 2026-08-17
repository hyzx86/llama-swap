import { parseDocument, Scalar, YAMLMap, YAMLSeq, type Document, type Node } from "yaml";

/**
 * Options that keep the serialized document as close to the user's original
 * formatting as possible: no padding inside flow collections (`[a, b]` stays
 * `[a, b]`) and no line-wrapping of long scalars. Comments and block scalars
 * (multi-line `cmd:`) are preserved by the Document API itself.
 */
export const SERIALIZE_OPTIONS = {
  flowCollectionPadding: false,
  lineWidth: 0,
} as const;

/** A node in the in-memory view of the YAML document. */
export interface ConfigNode {
  /** Path from the document root, e.g. ["models", "demo", "ttl"]. */
  path: string[];
  /** Key name for map entries; index (stringified) for sequence items. */
  key: string;
  kind: "map" | "seq" | "scalar";
  /** Display value for scalars; empty for collections. */
  value: string;
  /** 1-based source line of the key (or the scalar). */
  line: number;
  /** Child entries for collections. */
  children: ConfigNode[];
  /** True when the scalar holds a multi-line string (render as textarea). */
  multiline: boolean;
  /** Parsed scalar type, used to pick the right form control. */
  scalarType: "string" | "number" | "boolean" | "null" | "other";
}

/**
 * 1-based line number for a source offset. The document text is computed
 * once per buildTree call and shared via a WeakMap so the O(n) scan is not
 * repeated for every node.
 */
function lineOf(doc: Document, offset: number): number {
  const text = lineTextCache.get(doc) ?? doc.toString();
  lineTextCache.set(doc, text);
  let line = 1;
  const end = Math.min(offset, text.length);
  for (let i = 0; i < end; i++) {
    if (text.charCodeAt(i) === 10) line++;
  }
  return line;
}

const lineTextCache = new WeakMap<Document, string>();

function scalarDisplay(node: Node | null): string {
  if (!(node instanceof Scalar)) return "";
  const v = node.value;
  return v === null || v === undefined ? "" : String(v);
}

function isMultiline(node: Node | null): boolean {
  return node instanceof Scalar && (node.type === "BLOCK_LITERAL" || node.type === "BLOCK_FOLDED");
}

function buildNode(doc: Document, node: Node | null, path: string[], key: string): ConfigNode {
  if (node === null || node === undefined) {
    return { path, key, kind: "scalar", value: "", line: 1, children: [], multiline: false, scalarType: "null" };
  }
  if (node instanceof YAMLMap) {
    const children: ConfigNode[] = [];
    for (const pair of node.items) {
      const childKey = pair.key === null || pair.key === undefined ? "" : String(pair.key);
      children.push(buildNode(doc, pair.value as Node | null, [...path, childKey], childKey));
    }
    const range = node.range ?? [0, 0, 0];
    return { path, key, kind: "map", value: "", line: lineOf(doc, range[0]), children, multiline: false, scalarType: "other" };
  }
  if (node instanceof YAMLSeq) {
    const children: ConfigNode[] = [];
    for (let i = 0; i < node.items.length; i++) {
      children.push(buildNode(doc, node.items[i] as Node | null, [...path, String(i)], String(i)));
    }
    const range = node.range ?? [0, 0, 0];
    return { path, key, kind: "seq", value: "", line: lineOf(doc, range[0]), children, multiline: false, scalarType: "other" };
  }
  return {
    path,
    key,
    kind: "scalar",
    value: scalarDisplay(node),
    line: node.range ? lineOf(doc, node.range[0]) : 1,
    children: [],
    multiline: isMultiline(node),
    scalarType: scalarTypeOf(node),
  };
}

function scalarTypeOf(node: Node | null): ConfigNode["scalarType"] {
  if (!(node instanceof Scalar)) return "null";
  if (typeof node.value === "boolean") return "boolean";
  if (typeof node.value === "number") return "number";
  if (typeof node.value === "string") return "string";
  return "other";
}

/** Parse raw YAML into a Document (single source of truth for the editor). */
export function parseConfig(text: string): Document {
  const doc = parseDocument(text);
  return doc;
}

/** Build the tree model for the UI. Returns an empty array for empty docs. */
export function buildTree(doc: Document): ConfigNode[] {
  // Edits invalidate any cached serialization of this document.
  lineTextCache.delete(doc);
  if (!doc.contents) return [];
  const root = buildNode(doc, doc.contents, [], "");
  // The root itself is a map; flatten its top-level entries so the tree
  // starts at the first level (logLevel, models, peers, ...).
  if (root.kind === "map") return root.children;
  return [root];
}

/** Build the ConfigNode view for the node at `path` (null if not found). */
export function getNode(doc: Document, path: string[]): ConfigNode | null {
  const located = locate(doc, path);
  if (!located) return null;
  return buildNode(doc, located.node, path, path.length ? path[path.length - 1] : "");
}

/** Locate the live YAML node (and its parent) for a given path. */
export function locate(doc: Document, path: string[]): { node: Node | null; parent: Node | null; key: string } | null {
  let current: Node | null = doc.contents ?? null;
  let parent: Node | null = null;
  let lastKey = "";
  for (const segment of path) {
    if (current === null || current === undefined) return null;
    if (current instanceof YAMLMap) {
      const pair = current.items.find((p) => String(p.key) === segment);
      if (!pair) return null;
      parent = current;
      lastKey = segment;
      current = (pair.value as Node | null) ?? null;
    } else if (current instanceof YAMLSeq) {
      const idx = Number(segment);
      if (!Number.isInteger(idx) || idx < 0 || idx >= current.items.length) return null;
      parent = current;
      lastKey = segment;
      current = (current.items[idx] as Node | null) ?? null;
    } else {
      return null;
    }
  }
  return { node: current, parent, key: lastKey };
}

/** True when the node is a map or sequence collection. */
export function isCollection(node: Node | null | undefined): node is YAMLMap | YAMLSeq {
  return node instanceof YAMLMap || node instanceof YAMLSeq;
}

/** Serialize the document back to YAML text with the lossless options. */
export function serialize(doc: Document): string {
  return doc.toString(SERIALIZE_OPTIONS);
}

/**
 * Set a scalar value in place. Preserves the existing scalar type when the
 * new text parses to the same kind; multi-line strings use a literal block.
 */
export function setScalarValue(doc: Document, path: string[], text: string): void {
  const located = locate(doc, path);
  if (!located) return;
  let scalar = located.node as Scalar | null;
  if (scalar === null || scalar === undefined || scalar.type === undefined) {
    // A freshly added entry holds a null value; give it a plain scalar node.
    scalar = new Scalar("");
    if (located.parent instanceof YAMLMap) {
      const pair = located.parent.items.find((p) => String(p.key) === located.key);
      if (pair) pair.value = scalar;
    } else if (located.parent instanceof YAMLSeq) {
      const idx = Number(located.key);
      if (Number.isInteger(idx) && idx >= 0 && idx < located.parent.items.length) {
        located.parent.items[idx] = scalar;
      }
    }
  }

  const wasMultiline = isMultiline(scalar);
  const trimmed = text.replace(/\s+$/, "");

  if (wasMultiline || trimmed.includes("\n")) {
    scalar.type = "BLOCK_LITERAL";
    scalar.value = trimmed;
    return;
  }

  // Single-line: keep booleans and numbers typed when the text still parses
  // to that kind, otherwise fall back to a plain string.
  if (typeof scalar.value === "boolean") {
    if (text === "true") scalar.value = true;
    else if (text === "false") scalar.value = false;
    else {
      scalar.type = "PLAIN";
      scalar.value = text;
    }
    return;
  }
  if (typeof scalar.value === "number") {
    const n = Number(text);
    if (text.trim() !== "" && Number.isFinite(n)) {
      scalar.value = n;
    } else {
      scalar.type = "PLAIN";
      scalar.value = text;
    }
    return;
  }
  scalar.type = "PLAIN";
  scalar.value = text;
}

/** Add a new map entry with an empty key and null value; returns its path. */
export function addMapEntry(doc: Document, path: string[]): string[] {
  const located = locate(doc, path);
  if (!located || !(located.node instanceof YAMLMap)) return path;
  const map = located.node;
  if (map.flow) map.flow = false; // convert `{}` to block style
  map.set("", null);
  return [...path, ""];
}

/** Add a new sequence item with a null value; returns its path. */
export function addSeqItem(doc: Document, path: string[]): string[] {
  const located = locate(doc, path);
  if (!located || !(located.node instanceof YAMLSeq)) return path;
  const seq = located.node;
  if (seq.flow) seq.flow = false; // convert `[]` to block style
  seq.add(null);
  return [...path, String(seq.items.length - 1)];
}

/** Remove the node at `path` from its parent collection. */
export function removeNode(doc: Document, path: string[]): void {
  if (path.length === 0) return;
  const located = locate(doc, path);
  if (!located || !located.parent) return;
  if (located.parent instanceof YAMLMap) {
    const idx = located.parent.items.findIndex((p) => String(p.key) === located.key);
    if (idx >= 0) located.parent.items.splice(idx, 1);
  } else if (located.parent instanceof YAMLSeq) {
    const idx = Number(located.key);
    if (Number.isInteger(idx) && idx >= 0 && idx < located.parent.items.length) {
      located.parent.items.splice(idx, 1);
    }
  }
}

/** Rename a map key in place (only valid for map entries). */
export function renameMapKey(doc: Document, path: string[], newKey: string): void {
  if (path.length === 0) return;
  const located = locate(doc, path);
  if (!located || !(located.parent instanceof YAMLMap)) return;
  const pair = located.parent.items.find((p) => String(p.key) === located.key);
  if (pair) pair.key = newKey;
}
