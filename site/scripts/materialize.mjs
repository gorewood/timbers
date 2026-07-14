import {
  copyFileSync,
  existsSync,
  globSync,
  lstatSync,
  mkdirSync,
  readFileSync,
  realpathSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const defaultSiteRoot = path.resolve(scriptDir, "..");
const reservedRoutes = new Set(["assets", "content", "node_modules", "scripts", "_data", "_includes", "_site"]);

function within(parent, child) {
  const relative = path.relative(parent, child);
  return relative === "" || (!relative.startsWith("..") && !path.isAbsolute(relative));
}

function validateString(value, field) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${field} must be a non-empty string`);
  }
  return value.trim();
}

function validateCollection(collection, index) {
  const prefix = `collections[${index}]`;
  const id = validateString(collection.id, `${prefix}.id`);
  const route = validateString(collection.route, `${prefix}.route`).replace(/^\/+|\/+$/g, "");
  const kind = validateString(collection.kind, `${prefix}.kind`);

  if (!/^[a-z0-9][a-z0-9-]*$/.test(id)) throw new Error(`${prefix}.id must be lowercase kebab-case`);
  if (!/^[a-z0-9][a-z0-9/-]*$/.test(route) || route.split("/").some((part) => part === "." || part === "..")) {
    throw new Error(`${prefix}.route must be a safe relative URL path`);
  }
  if (reservedRoutes.has(route.split("/")[0])) throw new Error(`${prefix}.route uses reserved path ${route}`);
  if (kind !== "generated" && kind !== "native") throw new Error(`${prefix}.kind must be generated or native`);

  return {
    ...collection,
    id,
    route,
    kind,
    label: validateString(collection.label, `${prefix}.label`),
    root: validateString(collection.root, `${prefix}.root`),
    include: validateString(collection.include, `${prefix}.include`),
  };
}

function directoryData(collection) {
  return `export default {
  layout: "artifact.njk",
  tags: ["artifacts", "collection-${collection.id}"],
  artifact_kind: ${JSON.stringify(collection.kind)},
  collection_id: ${JSON.stringify(collection.id)},
  collection_label: ${JSON.stringify(collection.label)},
  collection_route: ${JSON.stringify(collection.route)},
  eleventyComputed: {
    permalink: ({ page }) => {
      const marker = ${JSON.stringify(`.generated/${collection.route}/`)};
      const inputPath = page.inputPath.replaceAll("\\\\", "/");
      const markerIndex = inputPath.indexOf(marker);
      if (markerIndex < 0) throw new Error("artifact is outside its staged collection: " + page.inputPath);
      if (inputPath.endsWith("/index.njk")) return ${JSON.stringify(`/${collection.route}/index.html`)};
      const relative = inputPath.slice(markerIndex + marker.length).replace(/\\.md$/, "");
      return ${JSON.stringify(`/${collection.route}/`)} + relative + "/index.html";
    },
  },
};
`;
}

function collectionIndex(collection) {
  return `---
layout: collection.njk
title: ${JSON.stringify(collection.label)}
description: ${JSON.stringify(collection.description ?? "")}
collection_tag: collection-${collection.id}
artifact_kind: ${JSON.stringify(collection.kind)}
permalink: /${collection.route}/index.html
eleventyExcludeFromCollections: true
---
`;
}

export function materialize(options = {}) {
  const siteRoot = path.resolve(options.siteRoot ?? defaultSiteRoot);
  const repoRoot = path.resolve(options.repoRoot ?? path.join(siteRoot, ".."));
  const configPath = path.resolve(options.configPath ?? path.join(siteRoot, "timbermill.json"));
  const generatedRoot = path.join(siteRoot, ".generated");
  const outputRoot = path.join(siteRoot, "_site");
  const config = JSON.parse(readFileSync(configPath, "utf8"));

  if (!Array.isArray(config.collections) || config.collections.length === 0) {
    throw new Error("timbermill.json must define at least one collection");
  }

  const repoReal = realpathSync(repoRoot);
  const ids = new Set();
  const routes = new Set();
  const outputs = new Set();
  const collections = config.collections.map(validateCollection);

  for (const collection of collections) {
    if (ids.has(collection.id)) throw new Error(`duplicate collection id: ${collection.id}`);
    if ([...routes].some((route) => collection.route === route || collection.route.startsWith(`${route}/`) || route.startsWith(`${collection.route}/`))) {
      throw new Error(`duplicate or overlapping collection route: ${collection.route}`);
    }
    ids.add(collection.id);
    routes.add(collection.route);
  }

  rmSync(generatedRoot, { recursive: true, force: true });
  rmSync(outputRoot, { recursive: true, force: true });
  mkdirSync(generatedRoot, { recursive: true });

  let count = 0;
  for (const collection of collections) {
    const root = path.resolve(repoRoot, collection.root);
    if (!within(repoRoot, root) || !existsSync(root)) throw new Error(`collection ${collection.id} root is outside the repository or missing`);
    const rootReal = realpathSync(root);
    if (!within(repoReal, rootReal)) throw new Error(`collection ${collection.id} root resolves outside the repository`);

    const destination = path.join(generatedRoot, collection.route);
    mkdirSync(destination, { recursive: true });
    writeFileSync(path.join(destination, `${path.basename(collection.route)}.11tydata.js`), directoryData(collection));
    writeFileSync(path.join(destination, "index.njk"), collectionIndex(collection));

    for (const relative of globSync(collection.include, { cwd: root })) {
      const source = path.resolve(root, relative);
      if (path.extname(relative).toLowerCase() !== ".md" || !lstatSync(source).isFile()) continue;
      const sourceReal = realpathSync(source);
      if (!within(rootReal, sourceReal) || !within(repoReal, sourceReal)) {
        throw new Error(`collection ${collection.id} matched a file outside its root: ${relative}`);
      }

      const identity = `${collection.route}/${relative.slice(0, -path.extname(relative).length)}`;
      if (identity === collection.route || outputs.has(identity)) throw new Error(`duplicate artifact route: ${identity}`);
      outputs.add(identity);

      const target = path.join(destination, relative);
      mkdirSync(path.dirname(target), { recursive: true });
      copyFileSync(source, target);
      count += 1;
    }
  }

  return { collections: collections.length, artifacts: count, generatedRoot };
}

if (path.resolve(process.argv[1] ?? "") === fileURLToPath(import.meta.url)) {
  const result = materialize();
  console.log(`Materialized ${result.artifacts} artifacts from ${result.collections} collections.`);
}
