import { title } from "@/components/primitives";
import { useEffect } from "react";

async function loadMarked() {
  // const styleUrl = `https://cdn.jsdelivr.net/npm/github-markdown-css@5/github-markdown-${theme}.min.css`;
  const styleUrl = `https://cdn.jsdelivr.net/npm/github-markdown-css@5/github-markdown.min.css`;
  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = styleUrl;
  document.head.appendChild(link);
  // @ts-ignore
  const { marked } = await import("https://cdn.jsdelivr.net/npm/marked/lib/marked.esm.js");
  const mdUrl = "https://raw.githubusercontent.com/YangZxi/Linkit/refs/heads/main/README.md";
  const resp = await fetch(mdUrl);
  let mdText = await resp.text();
  const imgPrefix = "https://github.com/YangZxi/Linkit/raw/main/";
  // @ts-ignore
  mdText = mdText.replaceAll("](images/", `](${imgPrefix}images/`);
  document.getElementById('content')!.innerHTML =
    marked.parse(mdText);
}

export default function AboutPage() {

  useEffect(() => {
    loadMarked();
  }, []);

  return (
    <section className="flex flex-col items-center justify-center gap-4 py-8 md:py-10">
      <div className="inline-block max-w-[720px] text-center justify-center">
        <h1 className={title()}>About</h1>
        <div id="content" className="markdown-body mt-3 text-left"></div>
      </div>
    </section>
  );
}
