import assert from "node:assert/strict";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { materialize } from "./materialize.mjs";

function fixture() {
  const repoRoot = mkdtempSync(path.join(os.tmpdir(), "timbermill-"));
  const siteRoot = path.join(repoRoot, "site");
  const reports = path.join(repoRoot, "docs", "reports");
  mkdirSync(path.join(reports, "nested"), { recursive: true });
  mkdirSync(siteRoot, { recursive: true });
  writeFileSync(path.join(reports, "one.md"), "---\ntitle: One\n---\n\nFirst.\n");
  writeFileSync(path.join(reports, "nested", "two.md"), "---\ntitle: Two\n---\n\nSecond.\n");
  writeFileSync(path.join(reports, "skip.txt"), "skip\n");
  const config = {
    collections: [{ id: "reports", label: "Reports", kind: "native", root: "docs/reports", include: "**/*", route: "reports" }],
  };
  const configPath = path.join(siteRoot, "timbermill.json");
  writeFileSync(configPath, JSON.stringify(config));
  return { repoRoot, siteRoot, reports, config, configPath };
}

test("materializes Markdown with stable relative paths and cleans stale output", () => {
  const item = fixture();
  const first = materialize(item);
  assert.equal(first.artifacts, 2);
  assert.equal(readFileSync(path.join(item.siteRoot, ".generated", "reports", "nested", "two.md"), "utf8"), "---\ntitle: Two\n---\n\nSecond.\n");
  assert.match(
    readFileSync(path.join(item.siteRoot, ".generated", "reports", "reports.11tydata.js"), "utf8"),
    /collection_route: "reports"[\s\S]*eleventyComputed:/,
  );
  assert.equal(existsSync(path.join(item.siteRoot, ".generated", "reports", "skip.txt")), false);

  writeFileSync(path.join(item.siteRoot, ".generated", "stale.md"), "stale");
  mkdirSync(path.join(item.siteRoot, "_site"));
  writeFileSync(path.join(item.siteRoot, "_site", "stale.html"), "stale");
  materialize(item);
  assert.equal(existsSync(path.join(item.siteRoot, ".generated", "stale.md")), false);
  assert.equal(existsSync(path.join(item.siteRoot, "_site", "stale.html")), false);
});

test("rejects roots outside the repository", () => {
  const item = fixture();
  item.config.collections[0].root = "../outside";
  writeFileSync(item.configPath, JSON.stringify(item.config));
  assert.throws(() => materialize(item), /outside the repository or missing/);
});

test("rejects duplicate collection routes", () => {
  const item = fixture();
  item.config.collections.push({ ...item.config.collections[0], id: "more-reports" });
  writeFileSync(item.configPath, JSON.stringify(item.config));
  assert.throws(() => materialize(item), /duplicate or overlapping collection route/);
});

test("rejects overlapping collection routes", () => {
  const item = fixture();
  item.config.collections.push({ ...item.config.collections[0], id: "more-reports", route: "reports/archive" });
  writeFileSync(item.configPath, JSON.stringify(item.config));
  assert.throws(() => materialize(item), /duplicate or overlapping collection route/);
});
