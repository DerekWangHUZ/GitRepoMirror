// 平台徽标：迁移自 src/ui/icons.js 的内联 SVG，改为 React JSX。
// github / gitlab 使用 fill 填充（沿用旧 .repo-icon.github svg 的规则）。

const GLYPHS = {
  git: (
    <>
      <path d="m12 2 10 10-10 10L2 12z" />
      <circle cx="8.5" cy="8.5" r="1.5" />
      <circle cx="15.5" cy="15.5" r="1.5" />
      <path d="m9.6 9.6 4.8 4.8M8.5 10v5" />
    </>
  ),
  github: (
    <path d="M12 .7a11.5 11.5 0 0 0-3.64 22.4c.58.1.79-.25.79-.56v-2.23c-3.22.7-3.9-1.37-3.9-1.37-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.7.08-.7 1.16.08 1.78 1.2 1.78 1.2 1.04 1.77 2.72 1.26 3.38.96.1-.75.4-1.26.74-1.55-2.57-.29-5.27-1.28-5.27-5.68 0-1.26.45-2.28 1.19-3.08-.12-.29-.52-1.46.11-3.04 0 0 .97-.31 3.16 1.18a10.97 10.97 0 0 1 5.76 0c2.2-1.49 3.16-1.18 3.16-1.18.63 1.58.23 2.75.11 3.04.74.8 1.19 1.82 1.19 3.08 0 4.41-2.7 5.38-5.28 5.67.42.36.79 1.07.79 2.16v3.2c0 .31.21.67.8.56A11.5 11.5 0 0 0 12 .7Z" />
  ),
  gitlab: <path d="m12 21 8.5-6.2L18.2 6h-3.1L12 15 8.9 6H5.8l-2.3 8.8z" />,
  gitee: (
    <>
      <circle cx="12" cy="12" r="9" />
      <path d="M21 12h-9v4h5" />
    </>
  ),
  codeberg: (
    <>
      <circle cx="12" cy="12" r="9" />
      <path d="m12 6.5 5.5 9h-11z" />
    </>
  ),
  azuredevops: <path d="M4 6h9l7 6-7 6H4l6-6z" />,
  generic: (
    <>
      <circle cx="6" cy="6" r="2" />
      <circle cx="18" cy="6" r="2" />
      <circle cx="12" cy="18" r="2" />
      <path d="M7.7 7.1 11 16M16.3 7.1 13 16M8 6h8" />
    </>
  ),
};

const FILLED = new Set(["github", "gitlab"]);

export function PlatformIcon({ platform, size = 18, className }) {
  const glyph = GLYPHS[platform] || GLYPHS.generic;
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      className={className}
      aria-hidden="true"
      fill={FILLED.has(platform) ? "currentColor" : "none"}
      stroke={FILLED.has(platform) ? "none" : "currentColor"}
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {glyph}
    </svg>
  );
}

export function GitMark({ size = 17, className }) {
  return <PlatformIcon platform="git" size={size} className={className} />;
}
