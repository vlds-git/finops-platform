/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://api-gateway:8080/api/:path*',
      },
    ];
  },
  images: {
    unoptimized: true,
  },
};

module.exports = nextConfig;
