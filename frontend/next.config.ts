import type {NextConfig} from "next";

const isExport = process.env.NEXT_STANDALONE_EXPORT === 'true';

const nextConfig: NextConfig = {
  reactCompiler: true,
  experimental: {
  },
  ...(isExport ? { output: 'export' } : {
    async rewrites() {
      const backendUrl = process.env.WAVELET_BACKEND_URL || 'http://localhost:8000';
      return [
        // 上传文件静态资源
        {
          source: '/f/:id',
          destination: `${ backendUrl }/f/:id`,
        },
        // robots.txt 路由代理到后端动态接口
        {
          source: '/robots.txt',
          destination: `${ backendUrl }/robots.txt`,
        },
        // 标准 RESTful API 接口
        {
          source: '/api/:path*',
          destination: `${ backendUrl }/api/:path*`,
        }
      ];
    },
  })
};

export default nextConfig;
