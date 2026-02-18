import { defineConfig } from "vite";
import { resolve } from "path";
import dts from "vite-plugin-dts";

export default defineConfig({
  build: {
    lib: {
      entry: resolve(__dirname, "src/index.ts"),
      name: "MirSDK",
      formats: ["es"],
      fileName: "index",
    },
    rollupOptions: {
      external: ["@bufbuild/protobuf", "nats.ws", "fzstd"],
      output: {
        preserveModules: true,
        preserveModulesRoot: "src",
        entryFileNames: "[name].js",
      },
    },
    sourcemap: true,
    target: "es2022",
    minify: false,
  },
  plugins: [
    dts({
      insertTypesEntry: true,
      rollupTypes: false,
      outDir: "dist",
    }),
  ],
  test: {
    globals: true,
    environment: "node",
  },
});
