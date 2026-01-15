import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import path from "node:path";
import fs from "node:fs";

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  // Determine interfaces package path (different in Docker vs local dev)
  // In Docker, the package is at ./packages/interfaces
  // Locally, it's at ../../packages/interfaces
  const dockerPath = path.resolve(__dirname, "./packages/interfaces");
  const localPath = path.resolve(__dirname, "../../packages/interfaces");
  const interfacesBasePath = fs.existsSync(dockerPath) ? dockerPath : localPath;

  return {
    plugins: [
      TanStackRouterVite(),
      react(),
      tailwindcss(),
      // Custom resolver for @janhq/interfaces package
      {
        name: "resolve-janhq-interfaces",
        resolveId(source: string) {
          if (source.startsWith("@janhq/interfaces/")) {
            const subPath = source.replace("@janhq/interfaces/", "");

            // Handle hooks paths
            if (subPath.startsWith("hooks/")) {
              const hookFile = path.resolve(
                interfacesBasePath,
                "src",
                subPath + ".ts",
              );
              if (fs.existsSync(hookFile)) {
                return hookFile;
              }
            }

            // Handle svgs paths
            if (subPath.startsWith("svgs/")) {
              const svgFile = path.resolve(
                interfacesBasePath,
                "src",
                subPath + ".tsx",
              );
              if (fs.existsSync(svgFile)) {
                return svgFile;
              }
            }

            // Handle ai-elements paths
            if (subPath.startsWith("ai-elements/")) {
              const aiFile = path.resolve(
                interfacesBasePath,
                "src",
                subPath + ".tsx",
              );
              if (fs.existsSync(aiFile)) {
                return aiFile;
              }
            }

            // Handle lib path
            if (subPath === "lib") {
              return path.resolve(interfacesBasePath, "src/lib/utils.ts");
            }

            // Handle constants path
            if (subPath === "constants") {
              return path.resolve(interfacesBasePath, "src/lib/constants.ts");
            }

            // Handle ui/ or components/ prefixed paths (strip the prefix)
            if (
              subPath.startsWith("ui/") ||
              subPath.startsWith("components/")
            ) {
              const componentName = subPath.replace(/^(ui|components)\//, "");
              const uiFile = path.resolve(
                interfacesBasePath,
                "src/ui",
                componentName + ".tsx",
              );
              if (fs.existsSync(uiFile)) {
                return uiFile;
              }
            }

            // Default: map directly to ui folder (for dialog, button, etc.)
            const uiFile = path.resolve(
              interfacesBasePath,
              "src/ui",
              subPath + ".tsx",
            );
            if (fs.existsSync(uiFile)) {
              return uiFile;
            }
          }
          return null;
        },
      },
    ],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
      dedupe: ["react", "react-dom"],
    },
    define: {
      JAN_API_BASE_URL: JSON.stringify(env.JAN_API_BASE_URL),
      VITE_GA_ID: JSON.stringify(env.VITE_GA_ID),
      VITE_AUTH_URL: JSON.stringify(
        env.VITE_AUTH_URL || "https://auth-dev.menlo.ai",
      ),
      VITE_AUTH_REALM: JSON.stringify(env.VITE_AUTH_REALM || "jan"),
      VITE_AUTH_CLIENT_ID: JSON.stringify(
        env.VITE_AUTH_CLIENT_ID || "jan-client",
      ),
      VITE_OAUTH_REDIRECT_URI: JSON.stringify(
        env.VITE_OAUTH_REDIRECT_URI || "http://localhost:3001/auth/callback",
      ),
      VITE_REPORT_ISSUE_URL: JSON.stringify(env.VITE_REPORT_ISSUE_URL || "/"),
      BROWSER_SERVER_NAME: JSON.stringify("Jan Browser Extension"),
      EXTENSION_ID: JSON.stringify(
        env.EXTENSION_ID || "mkciifcjehgnpaigoiaakdgabbpfppal",
      ),
      CHROME_STORE_URL: JSON.stringify(
        env.CHROME_STORE_URL ||
          "https://chromewebstore.google.com/detail/jan-browser-mcp/mkciifcjehgnpaigoiaakdgabbpfppal",
      ),
    },
  };
});
