import path from 'node:path';
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  output: 'standalone',
  outputFileTracingRoot: path.resolve(__dirname),
  serverExternalPackages: ['wechatpay-node-v3'],
  turbopack: {
    root: path.resolve(__dirname),
  },
};

export default nextConfig;
