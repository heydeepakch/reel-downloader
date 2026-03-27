/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() {
    // In local dev, proxy /api to Go backend on :8080 so fetch('/api/reel') works
    if (process.env.NODE_ENV === "development") {
      return [
        { source: "/api/:path*", destination: "http://localhost:8080/api/:path*" },
      ];
    }
    return [];
  },
};

export default nextConfig;
