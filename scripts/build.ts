
import solidPlugin from "@opentui/solid/bun-plugin"

const result = await Bun.build({
  entrypoints: ["src/cli.tsx", "src/workers/fuzzy.ts"],
  outdir: "dist",
  target: "bun",
  plugins: [solidPlugin],
  external: ["@opentui/core", "shiki"],
  sourcemap: "external",
})
if (!result.success) throw new AggregateError(result.logs, "Daisy build failed")
