export const siteConfig = {
  name: "Linkit",
  description: "Upload once, use anywhere.",
  navItems: [
    { label: "Gallery", href: "/gallery", mobile: true },
    // { label: "ClipSync", href: "/clipsync" },
    { label: "About", href: "/about" },
  ],
};
export type SiteConfig = typeof siteConfig;
