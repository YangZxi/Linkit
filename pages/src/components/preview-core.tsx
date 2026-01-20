import { useEffect, useRef, useState } from "react";
import { Image, Spinner, Textarea } from "@heroui/react";
import clsx from "clsx";

import { MediaType } from "@/lib/file";

const PDFJS_CDN = "https://unpkg.com/pdfjs-dist@5.4.449/build/pdf.min.mjs";
const PDFJS_WORKER_CDN = "https://unpkg.com/pdfjs-dist@5.4.449/build/pdf.worker.min.mjs";
// const PDFJS_CSS_CDN = "https://cdnjs.cloudflare.com/ajax/libs/pdf.js/5.4.149/pdf_viewer.min.css";
const PDFJS_SCRIPT_ID = "pdfjs-lib-script";

declare global {
  interface Window {
    pdfjsLib?: any;
  }
}

let pdfjsPromise: Promise<any> | null = null;

// 通过 script 标签按需加载 pdf.js，避免打包期动态 import 警告
function loadPdfJs(): Promise<any> {
  if (typeof window === "undefined") {
    return Promise.reject(new Error("仅浏览器环境可加载 pdf.js"));
  }
  if (window.pdfjsLib) {
    return Promise.resolve(window.pdfjsLib);
  }
  if (pdfjsPromise) return pdfjsPromise;

  pdfjsPromise = new Promise((resolve, reject) => {
    const existing = document.getElementById(PDFJS_SCRIPT_ID) as
      | HTMLScriptElement
      | null;
    if (existing) {
      existing.addEventListener(
        "load",
        () => {
          if (window.pdfjsLib) {
            resolve(window.pdfjsLib);
          } else {
            reject(new Error("pdf.js 加载失败"));
          }
        },
        { once: true },
      );
      existing.addEventListener(
        "error",
        () => reject(new Error("pdf.js 加载失败")),
        { once: true },
      );
      return;
    }

    const script = document.createElement("script");
    script.id = PDFJS_SCRIPT_ID;
    script.src = PDFJS_CDN;
    script.async = true;
    script.type = "module";
    script.onload = () => {
      if (window.pdfjsLib) {
        resolve(window.pdfjsLib);
      } else {
        reject(new Error("pdf.js 加载失败"));
      }
    };
    script.onerror = () => reject(new Error("pdf.js 加载失败"));
    document.body.appendChild(script);

    // const style = document.createElement("link");
    // style.rel = "stylesheet";
    // style.href = PDFJS_CSS_CDN;
    // document.head.appendChild(style);
  });

  return pdfjsPromise;
}

type PreviewCoreProps = {
  type: MediaType;
  filename: string;
  className?: string;
  style?: React.CSSProperties;
  rawUrl: string;
};

/**
 * PDF 预览渲染，按需动态加载 pdf.js，避免非 PDF 时的额外体积。
 */
function PdfPreview({
  className,
  style,
  rawUrl,
}: Pick<PreviewCoreProps, "className" | "style" | "rawUrl">) {
  const pdfContainerRef = useRef<HTMLDivElement>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let canceled = false;

    // 进入 PDF 预览时清空旧内容，避免切换类型时残留。
    if (pdfContainerRef.current) {
      pdfContainerRef.current.innerHTML = "";
    }

    setLoading(true);
    setError(null);

    (async () => {
      try {
        const pdfjsLib: any = await loadPdfJs();
        console.log("pdfjsLib", pdfjsLib);
        pdfjsLib.GlobalWorkerOptions.workerSrc = PDFJS_WORKER_CDN;

        const loadingTask = pdfjsLib.getDocument(rawUrl);
        const pdf = await loadingTask.promise;

        if (canceled) return;

        const pdfDom = pdfContainerRef.current;
        if (!pdfDom) return;

        for (let pageIndex = 1; pageIndex <= pdf.numPages; pageIndex++) {
          const page = await pdf.getPage(pageIndex);
          if (canceled) return;

          const viewport = page.getViewport({ scale: 1.1 });
          const canvas = document.createElement("canvas");
          const context = canvas.getContext("2d");

          canvas.width = viewport.width;
          canvas.height = viewport.height;
          canvas.className =
            "mb-4 last:mb-0 rounded-xl bg-white shadow-sm dark:bg-default-50/20";

          await page.render({ canvasContext: context!, viewport }).promise;
          if (canceled) return;

          pdfDom.appendChild(canvas);
        }
      } catch (err) {
        if (!canceled) {
          setError("PDF 加载失败，请稍后重试");
        }
      } finally {
        if (!canceled) {
          setLoading(false);
        }
      }
    })();

    return () => {
      canceled = true;
      if (pdfContainerRef.current) {
        pdfContainerRef.current.innerHTML = "";
      }
    };
  }, [rawUrl]);

  return (
    <div
      className={clsx(
        "relative flex flex-1 overflow-hidden rounded-2xl border border-default-200/70 bg-default-50/70 shadow-lg dark:border-default-100/20 dark:bg-default-50/10",
        className,
      )}
    >
      <div
        ref={pdfContainerRef}
        className="relative h-full w-full flex justify-center overflow-y-auto space-y-4 p-4"
      />
      {loading && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-background/70 backdrop-blur-sm">
          <Spinner color="primary" label="PDF 加载中" />
        </div>
      )}
      {error && (
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-background/80 p-4 text-danger-500 backdrop-blur-sm">
          <span>{error}</span>
          <span className="text-xs text-default-500">请检查文件链接后重试</span>
        </div>
      )}
    </div>
  );
}

function TextPreview({
  className,
  rawUrl,
}: Pick<PreviewCoreProps, "className" | "rawUrl">) {
  const [content, setContent] = useState<string>("Loading...");

  useEffect(() => {
    let isMounted = true;

    fetch(rawUrl)
      .then((res) => res.text())
      .then((text) => {
        if (isMounted) setContent(text);
      })
      .catch(() => {
        if (isMounted) setContent("加载文本失败");
      });

    return () => {
      isMounted = false;
    };
  }, [rawUrl]);

  return (
    <Textarea
      isReadOnly
      classNames={{
        input: className,
      }}
      maxRows={15}
      value={content}
    />
  );
}

function AudioPreview({
  className,
  rawUrl,
}: Pick<PreviewCoreProps, "className" | "rawUrl">) {
  return (
    <div
      className={clsx(
        className,
        "flex flex-col gap-3 rounded-2xl border border-default-200/70 bg-default-50/60 p-6 shadow-lg dark:border-default-100/20 dark:bg-default-50/10",
      )}
    >
      <p className="text-sm text-default-500">音频预览</p>
      <audio controls className="w-full" src={rawUrl}>
        <track kind="captions" label="No captions available" src="" />
      </audio>
    </div>
  );
}

function VideoPreview({
  className,
  rawUrl,
}: Pick<PreviewCoreProps, "className" | "rawUrl">) {
  return (
    <video
      controls
      className={clsx(
        className,
        "rounded-2xl bg-black shadow-lg outline-none",
      )}
      src={rawUrl}
    >
      <track kind="captions" label="No captions available" src="" />
    </video>
  );
}

function ImagePreview({
  className,
  style,
  rawUrl,
  filename,
}: Pick<PreviewCoreProps, "className" | "style" | "rawUrl" | "filename">) {
  return (
    <Image
      alt={filename}
      classNames={{
        wrapper: `${className} max-h-[460px]`,
        img: "object-contain"
      }}
      src={rawUrl}
    />
  );
}

function FallbackPreview({ className }: Pick<PreviewCoreProps, "className">) {
  return (
    <div
      className={clsx(
        className,
        "flex items-center justify-center rounded-2xl border border-default-200/70 bg-default-50/60 text-default-500 shadow-inner dark:border-default-100/20 dark:bg-default-50/10",
      )}
    >
      暂不支持该类型预览，请使用下载链接打开
    </div>
  );
}

function PreviewCore({
  type,
  filename,
  className,
  style,
  rawUrl,
}: PreviewCoreProps) {
  if (type === "image") {
    return (
      <ImagePreview className={className} filename={filename} rawUrl={rawUrl} />
    );
  }

  if (type === "video") {
    return <VideoPreview className={className} rawUrl={rawUrl} />;
  }

  if (type === "audio") {
    return <AudioPreview className={className} rawUrl={rawUrl} />;
  }

  if (type === "text") {
    return <TextPreview className={className} rawUrl={rawUrl} />;
  }

  if (type === "pdf") {
    return <PdfPreview className={className} rawUrl={rawUrl} />;
  }

  return <FallbackPreview className={className} />;
}

export { PreviewCore };
export default PreviewCore;
