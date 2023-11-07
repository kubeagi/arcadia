import { defineConfig } from "father";

export default defineConfig({
  umd: {
    name: "ArcadiaBffSDK",
    entry: "src",
    sourcemap: true,
    externals: {
      react: "var window.React",
      "react-dom": "var window.ReactDOM",
      "prop-types": "var window.PropTypes",
      antd: "var window.antd",
      "@tenx-ui/materials": "var window.TenxUiMaterials",
    },
  },
  esm: { input: "src" },
  // cjs: { input: 'src' },
  // prebundle: {},
});
