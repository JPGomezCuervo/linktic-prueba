import { fileURLToPath, URL } from "node:url";

import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import { loadEnv } from "vite";
import { defineConfig } from "vitest/config";
import tschecker from "vite-plugin-checker";
import vueDevTools from "vite-plugin-vue-devtools";

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), "");

	return {
		plugins: [
			vue(),
			vueDevTools(),
			tailwindcss(),
			tschecker({
				vueTsc: {
					tsconfigPath: "tsconfig.app.json",
				},
			}),
		],
		resolve: {
			alias: {
				"@": fileURLToPath(new URL("./src", import.meta.url)),
				"@components": fileURLToPath(new URL("./src/components", import.meta.url)),
				"@views": fileURLToPath(new URL("./src/views", import.meta.url)),
				"@composables": fileURLToPath(new URL("./src/composables", import.meta.url)),
				"@test": fileURLToPath(new URL("./src/test", import.meta.url)),
			},
		},
		server: {
			proxy: {
				"/api": {
					target: env.VITE_API_PROXY_TARGET || "http://localhost:8080",
					changeOrigin: true,
					rewrite: (path) => path.replace(/^\/api/, ""),
				},
			},
		},
		test: {
			globals: true,
			environment: "jsdom",
			setupFiles: ["src/test/setup.ts"],
		},
	};
});
