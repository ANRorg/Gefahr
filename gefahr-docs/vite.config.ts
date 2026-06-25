import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const repoName = process.env.GITHUB_REPOSITORY?.split("/")[1];
const base = process.env.DOCS_BASE ?? (process.env.GITHUB_ACTIONS && repoName ? `/${repoName}/` : "/");

export default defineConfig({
  base,
  plugins: [react()],
});
