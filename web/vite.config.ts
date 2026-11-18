import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath, URL } from "node:url";
import { readFileSync } from "node:fs";
import type { Plugin } from "vite";

function inlineTouchIcon(): Plugin {
	return {
		name: "inline-touch-icon",
		apply: "build",
		transformIndexHtml(html) {
			const png = readFileSync(new URL("./public/apple-touch-icon.png", import.meta.url)).toString("base64");
			return html.replace('href="/apple-touch-icon.png"', `href="data:image/png;base64,${png}"`);
		},
	};
}

export default defineConfig({
	plugins: [react(), inlineTouchIcon()],
	resolve: {
		dedupe: ["react", "react-dom", "react-router", "react-router-dom", "sonner"],
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
