import timbermill from "./timbermill.json" with { type: "json" };

const configuredPrefix = process.env.TIMBERMILL_PATH_PREFIX ?? timbermill.site.path_prefix ?? "/";
const pathPrefix = `/${configuredPrefix.replace(/^\/+|\/+$/g, "")}${configuredPrefix === "/" ? "" : "/"}`;

function siteUrl(value = "/") {
  const relative = String(value).replace(/^\/+/, "");
  return pathPrefix === "/" ? `/${relative}` : `${pathPrefix}${relative}`;
}

function displayDate(value) {
  if (!value) return "";
  const date = value instanceof Date ? value : new Date(`${value}T00:00:00Z`);
  if (Number.isNaN(date.valueOf())) return String(value);
  return new Intl.DateTimeFormat("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  }).format(date);
}

function isoDate(value) {
  if (!value) return "";
  const date = value instanceof Date ? value : new Date(`${value}T00:00:00Z`);
  return Number.isNaN(date.valueOf()) ? String(value) : date.toISOString();
}

function byNewest(items = []) {
  return [...items].sort((left, right) => right.date - left.date);
}

function take(items = [], count = 0) {
  return items.slice(0, count);
}

export default function (eleventyConfig) {
  eleventyConfig.addPassthroughCopy("assets");
  eleventyConfig.addFilter("siteUrl", siteUrl);
  eleventyConfig.addFilter("displayDate", displayDate);
  eleventyConfig.addFilter("isoDate", isoDate);
  eleventyConfig.addFilter("byNewest", byNewest);
  eleventyConfig.addFilter("take", take);
  eleventyConfig.addGlobalData("site", { ...timbermill.site, path_prefix: pathPrefix });
  eleventyConfig.addGlobalData("timbermill", timbermill);

  eleventyConfig.ignores.add("content/**");
  eleventyConfig.ignores.add("scripts/**");
  eleventyConfig.ignores.add("README.md");

  return {
    pathPrefix,
    dir: {
      input: ".",
      output: "_site",
      includes: "_includes",
      data: "_data",
    },
    markdownTemplateEngine: false,
    htmlTemplateEngine: "njk",
    templateFormats: ["md", "njk"],
  };
}
