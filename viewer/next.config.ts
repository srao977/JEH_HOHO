import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  agentRules: false,
  output: "standalone",
  generateBuildId: async () => "jeh-hoho-0.1.0-candidate",
  turbopack: {
    root: process.cwd(),
  },
};

export default nextConfig;
