const currentYear = new Date().getFullYear();

export const APP_CONFIG = {
  name: "ARTEX",
  // Injected at build time by next.config.mjs (env.NEXT_PUBLIC_APP_VERSION) so the
  // client bundle does not embed the whole package.json.
  version: process.env.NEXT_PUBLIC_APP_VERSION ?? "",
  copyright: `© ${currentYear}, ARTEX.`,
  meta: {
    title: "ARTEX — 自主渗透测试控制台",
    description: "LLM 驱动的自主渗透测试系统控制台",
  },
};
