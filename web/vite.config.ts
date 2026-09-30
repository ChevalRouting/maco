import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
	plugins: [react()],
	resolve: {
		dedupe: ["react", "react-dom", "react-router", "react-router-dom"],
		alias: {
			"@": fileURLToPath(new URL("./src", import.meta.url)),
		},
	},
	build: {
		target: ["es2022", "safari15"],
		outDir: "../pkg/api/dist",
		emptyOutDir: true,
	},
	optimizeDeps: {
		exclude: ["@novnc/novnc"],
	},
	server: {
		proxy: {
			"^/api/": {
				target: "http://localhost:8080",
				changeOrigin: false,
				ws: true,
			},
		},
	},
});
