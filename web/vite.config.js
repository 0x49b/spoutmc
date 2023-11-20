import react from "@vitejs/plugin-react";
import * as path from "path";

export default {
    // Other Vite configurations...
    plugins: [react()],
    resolve: {
        alias: {
            "@": path.resolve(__dirname, "src"),
        },
    },
    // Exclude node_modules from watch mode
    optimizeDeps: {
        exclude: ['node_modules'],
    },
};