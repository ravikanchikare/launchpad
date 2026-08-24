import { readdir, readFile, stat } from "node:fs/promises";
import { dirname, extname, join, resolve } from "node:path";

const dist = resolve(process.argv[2] || "frontend/dist");
const indexPath = join(dist, "index.html");

async function filesUnder(directory) {
  const result = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) result.push(...(await filesUnder(path)));
    else if (entry.isFile()) result.push(path);
  }
  return result;
}

const index = await readFile(indexPath, "utf8");
const files = await filesUnder(dist);
const relativeFiles = files.map((path) => path.slice(dist.length + 1));
if (!relativeFiles.some((path) => extname(path) === ".js")) {
  throw new Error(`frontend bundle has no JavaScript asset: ${dist}`);
}
if (!relativeFiles.some((path) => extname(path) === ".css")) {
  throw new Error(`frontend bundle has no CSS asset: ${dist}`);
}

for (const match of index.matchAll(/<(?:script|link|img)\b[^>]*(?:src|href)=["']([^"']+)["'][^>]*>/gi)) {
  const reference = match[1];
  if (/^(?:https?:)?\/\//i.test(reference)) {
    throw new Error(`frontend entry has an external runtime asset: ${reference}`);
  }
  if (/^(?:data:|#)/i.test(reference)) continue;
  const clean = reference.replace(/^\.\//, "").replace(/^\//, "").split(/[?#]/, 1)[0];
  if (!clean) continue;
  const target = resolve(dist, clean);
  if (!target.startsWith(`${dist}/`) || !(await stat(target)).isFile()) {
    throw new Error(`frontend entry references a missing bundled asset: ${reference}`);
  }
}

const forbidden = [
  /https?:\/\/(?:127\.0\.0\.1|localhost):5173\b/i,
  /\bNATIVE_SDK_FRONTEND_URL\b/,
  /(?:src|href)=["']\/?src\//i,
  /@vite\/client/i,
];
for (const path of files.filter((file) => [".html", ".js", ".css"].includes(extname(file)))) {
  const source = await readFile(path, "utf8");
  for (const pattern of forbidden) {
    if (pattern.test(source)) throw new Error(`development-only reference ${pattern} found in ${path}`);
  }
  if (extname(path) === ".css") {
    for (const match of source.matchAll(/url\(\s*["']?([^"')]+)["']?\s*\)/gi)) {
      const reference = match[1].trim();
      if (/^(?:https?:)?\/\//i.test(reference)) {
        throw new Error(`frontend stylesheet has an external runtime asset: ${reference}`);
      }
      if (/^(?:data:|#)/i.test(reference)) continue;
      const clean = reference.split(/[?#]/, 1)[0];
      const target = reference.startsWith("/")
        ? resolve(dist, clean.replace(/^\//, ""))
        : resolve(dirname(path), clean);
      if (!target.startsWith(`${dist}/`) || !(await stat(target)).isFile()) {
        throw new Error(`frontend stylesheet references a missing bundled asset: ${reference}`);
      }
    }
  }
}

console.log(`Verified offline frontend bundle: ${relativeFiles.length} files in ${dist}`);
