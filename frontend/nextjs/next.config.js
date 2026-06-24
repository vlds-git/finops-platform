/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: 'http://api-gateway:8080/api/v1/:path*',
      },
    ];
  },
}

module.exports = nextConfig
