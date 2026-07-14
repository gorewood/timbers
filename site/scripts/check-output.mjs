import { existsSync, globSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const siteRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = path.resolve(siteRoot, "..");
const outputRoot = path.join(siteRoot, "_site");
const config = JSON.parse(readFileSync(path.join(siteRoot, "timbermill.json"), "utf8"));
const collectionRoutes = config.collections.map(({ route }) => route.replace(/^\/+|\/+$/g, ""));
const configuredPrefix = process.env.TIMBERMILL_PATH_PREFIX ?? config.site.path_prefix ?? "/";
const prefix = `/${configuredPrefix.replace(/^\/+|\/+$/g, "")}${configuredPrefix === "/" ? "" : "/"}`;
const htmlFiles = globSync("**/*.html", { cwd: outputRoot });

let sourceCount = 0;
for (const collection of config.collections) {
  const root = path.resolve(repoRoot, collection.root);
  sourceCount += globSync(collection.include, { cwd: root }).filter((name) => path.extname(name).toLowerCase() === ".md").length;
}

const expectedPages = 1 + config.collections.length + sourceCount;
if (htmlFiles.length !== expectedPages) {
  throw new Error(`expected ${expectedPages} HTML pages, found ${htmlFiles.length}`);
}

const broken = [];
for (const file of htmlFiles) {
  const html = readFileSync(path.join(outputRoot, file), "utf8");
  const headingCount = [...html.matchAll(/<h1(?:\s|>)/g)].length;
  if (headingCount !== 1) broken.push(`${file}: expected one h1, found ${headingCount}`);
  const backLink = html.match(/class="back-link" href="([^"]+)"/);
  if (backLink) {
    const outputPath = file.split(path.sep).join("/");
    const collectionRoute = collectionRoutes.find((route) => outputPath.startsWith(`${route}/`) && outputPath !== `${route}/index.html`);
    if (!collectionRoute) {
      broken.push(`${file}: artifact does not belong to a configured collection route`);
      continue;
    }
    const expectedBackLink = `${prefix}${collectionRoute}/`;
    if (backLink[1] !== expectedBackLink) {
      broken.push(`${file}: artifact back link is ${backLink[1]}, expected ${expectedBackLink}`);
    }
  }
  for (const match of html.matchAll(/href="([^"]+)"/g)) {
    const href = match[1];
    if (/^(?:https?:|mailto:|#)/.test(href)) continue;
    if (!href.startsWith(prefix)) {
      broken.push(`${file}: internal URL does not use path prefix: ${href}`);
      continue;
    }

    const pathname = href.slice(prefix.length).split(/[?#]/, 1)[0];
    const target = pathname === "" || pathname.endsWith("/") ? path.join(pathname, "index.html") : pathname;
    if (!existsSync(path.join(outputRoot, target))) broken.push(`${file}: missing ${href}`);
  }
}

if (broken.length) throw new Error(`broken internal links:\n${broken.join("\n")}`);
console.log(`Verified ${htmlFiles.length} pages and their internal links under ${prefix}.`);
