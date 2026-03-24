import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  optimizeDeps: {
    exclude: [],
  },
  define: {
    // Ensure import.meta.env.PROD is properly set
    'import.meta.env.PROD': JSON.stringify(process.env.NODE_ENV === 'production')
  }
});