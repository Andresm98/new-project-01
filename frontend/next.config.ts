import type { NextConfig } from "next";

// Static export: the app is a pure SPA served by nginx (no Node runtime in prod).
const nextConfig: NextConfig = {
  output: "export",
  trailingSlash: true,
  images: { unoptimized: true },
  poweredByHeader: false,
  reactStrictMode: true,
};

export default nextConfig;
