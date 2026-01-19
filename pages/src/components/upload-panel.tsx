"use client";

import type { ChangeEvent, DragEvent, KeyboardEvent, SVGProps } from "react";

import { useCallback, useEffect, useRef, useState } from "react";
import clsx from "clsx";
import { addToast, Button, Image } from "@heroui/react";

import { UploadChunkResponse, UploadCompletedResponse } from "@/types/api";
import { Icon } from "@iconify/react";

type UploadType = "image" | "video" | "audio" | "other";
type UploadStatus = "idle" | "uploading" | "success" | "error" | "canceled";

type UploadItem = {
  id: string;
  file: File;
  type: UploadType;
  uploadId: string;
  progress: number;
  status: UploadStatus;
  previewUrl?: string;
  error?: string;
  controller?: AbortController;
  shareCode?: string;
  resourceId?: number;
};

const CHUNK_THRESHOLD = 100 * 1024 * 1024; // 100MB
const CHUNK_SIZE = 8 * 1024 * 1024; // 8MB

const UploadGridItem: UploadItem = {
  id: "upload",
  file: null as unknown as File,
  type: "other",
  uploadId: "",
  progress: 0,
  status: "idle",
}

function CloudArrowUpIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      aria-hidden="true"
      fill="none"
      height="1em"
      viewBox="0 0 24 24"
      width="1em"
      {...props}
    >
      <path
        d="M9 15l3-3m0 0l3 3m-3-3v6"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
      <path
        d="M6.5 18H18a4 4 0 001.66-7.62 5.5 5.5 0 00-10.78-1.62A4.5 4.5 0 006.5 18z"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
    </svg>
  );
}

function VideoIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      aria-hidden="true"
      fill="none"
      height="1em"
      viewBox="0 0 24 24"
      width="1em"
      {...props}
    >
      <rect
        height="14"
        rx="2"
        stroke="currentColor"
        strokeWidth="1.5"
        width="14"
        x="4"
        y="5"
      />
      <path
        d="M14 10l4-2.5v9L14 14v-4z"
        stroke="currentColor"
        strokeWidth="1.5"
      />
    </svg>
  );
}

function AudioIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      aria-hidden="true"
      fill="none"
      height="1em"
      viewBox="0 0 24 24"
      width="1em"
      {...props}
    >
      <path
        d="M7 15.5A2.5 2.5 0 109.5 18V7.5L17 5v9.5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
      <circle
        cx="17"
        cy="16.5"
        r="2.5"
        stroke="currentColor"
        strokeWidth="1.5"
      />
    </svg>
  );
}

function FileIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      aria-hidden="true"
      fill="none"
      height="1em"
      viewBox="0 0 24 24"
      width="1em"
      {...props}
    >
      <path
        d="M7 4h6l4 4v10a2 2 0 01-2 2H7a2 2 0 01-2-2V6a2 2 0 012-2z"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
      <path
        d="M13 4.5V8a.5.5 0 00.5.5H17"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
    </svg>
  );
}

function RetryIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      aria-hidden="true"
      fill="none"
      height="1em"
      viewBox="0 0 24 24"
      width="1em"
      {...props}
    >
      <path
        d="M20 11.5A7.5 7.5 0 105 9.7"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
      <path
        d="M5 4v6h6"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
    </svg>
  );
}

function resolveType(file: File): UploadType {
  if (file.type.startsWith("image/")) return "image";
  if (file.type.startsWith("video/")) return "video";
  if (file.type.startsWith("audio/")) return "audio";

  return "other";
}

function createUploadId(file: File) {
  const random = Math.random().toString(16).slice(2);

  return `${file.name}-${file.size}-${file.lastModified}-${random}`;
}

type UploadPanelProps = {};

export default function UploadPanel({ }: UploadPanelProps = {}) {
  const [items, setItems] = useState<UploadItem[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const previewUrlsRef = useRef<string[]>([]);
  const itemsRef = useRef<UploadItem[]>([]);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  useEffect(() => {
    itemsRef.current = items;
  }, [items]);

  // 清理图片预览地址，防止内存泄露
  const revokePreview = useCallback((url?: string) => {
    if (!url) return;
    URL.revokeObjectURL(url);
    previewUrlsRef.current = previewUrlsRef.current.filter((u) => u !== url);
  }, []);

  const buildItems = useCallback((files: File[]): UploadItem[] => {
    return files.map((file) => {
      const type = resolveType(file);
      const uploadId = createUploadId(file);
      const id = `${uploadId}-${Math.random().toString(16).slice(2)}`;

      if (type === "image") {
        const previewUrl = URL.createObjectURL(file);

        previewUrlsRef.current.push(previewUrl);

        return {
          id,
          file,
          type,
          previewUrl,
          uploadId,
          progress: 0,
          status: "idle",
        };
      }

      return {
        id,
        file,
        type,
        uploadId,
        progress: 0,
        status: "idle",
      };
    });
  }, []);

  const updateItem = useCallback(
    (id: string, updater: (item: UploadItem) => Partial<UploadItem>) => {
      setItems((prev) =>
        prev.map((item) =>
          item.id === id ? { ...item, ...updater(item) } : item,
        ),
      );
    },
    [],
  );

  const removeItem = useCallback(
    (id: string) => {
      setItems((prev) => {
        const target = prev.find((item) => item.id === id);

        revokePreview(target?.previewUrl);

        return prev.filter((item) => item.id !== id);
      });
    },
    [revokePreview],
  );

  const handleFiles = useCallback(
    (incoming: FileList | File[]) => {
      const files = Array.from(incoming);

      if (!files.length) return;
      const newItems = buildItems(files);

      setItems((prev) => [...prev, ...newItems]);
    },
    [buildItems],
  );

  const onDrop = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      event.preventDefault();
      setIsDragging(false);
      if (event.dataTransfer?.files?.length) {
        handleFiles(event.dataTransfer.files);
      }
    },
    [handleFiles],
  );

  const onDragOver = useCallback((event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setIsDragging(true);
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "copy";
    }
  }, []);

  const onDragLeave = useCallback(() => {
    setIsDragging(false);
  }, []);

  const onInputChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      if (event.target.files?.length) {
        handleFiles(event.target.files);
      }
      event.target.value = "";
    },
    [handleFiles],
  );

  const onBrowseClick = useCallback(() => {
    inputRef.current?.click();
  }, []);

  const onUploadKeyDown = useCallback(
    (event: KeyboardEvent<HTMLDivElement>) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        onBrowseClick();
      }
    },
    [onBrowseClick],
  );

  // 支持直接粘贴文件
  useEffect(() => {
    const onPaste = (event: ClipboardEvent) => {
      if (event.clipboardData?.files?.length) {
        handleFiles(event.clipboardData.files);
      }
    };

    window.addEventListener("paste", onPaste);

    return () => window.removeEventListener("paste", onPaste);
  }, [handleFiles]);

  // 组件卸载时释放 ObjectURL，避免内存占用
  useEffect(() => {
    return () => {
      previewUrlsRef.current.forEach((url) => URL.revokeObjectURL(url));
    };
  }, []);

  const fetchUploadedChunks = useCallback(
    async (uploadId: string, signal: AbortSignal) => {
      try {
        const res = await fetch(
          `/api/upload?uploadId=${encodeURIComponent(uploadId)}`,
          {
            method: "GET",
            signal,
          },
        );

        if (!res.ok) return new Set<number>();
        const json = await res.json();
        const list: number[] = json?.data?.uploaded ?? [];

        return new Set(list);
      } catch {
        return new Set<number>();
      }
    },
    [],
  );

  // 小文件单请求上传，使用 XHR 获得真实进度
  const uploadSingle = useCallback(
    (item: UploadItem, controller: AbortController) =>
      new Promise<void>((resolve, reject) => {
        const formData = new FormData();

        formData.append("file", item.file);
        formData.append("uploadId", item.uploadId);
        formData.append("fileName", item.file.name);
        formData.append("fileSize", `${item.file.size}`);

        const xhr = new XMLHttpRequest();

        xhr.open("POST", "/api/upload");
        xhr.responseType = "json";

        xhr.upload.onprogress = (event) => {
          if (!event.lengthComputable) return;
          const percent = Math.min(
            99,
            Math.round((event.loaded / event.total) * 100),
          );

          updateItem(item.id, () => ({
            progress: percent,
            status: "uploading",
          }));
        };

        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            const data = (xhr.response?.data ?? {}) as UploadCompletedResponse;

            updateItem(item.id, () => ({
              progress: 100,
              status: "success",
              shareCode: data?.shareCode,
              resourceId: data?.resourceId,
            }));
            resolve();
          } else {
            const msg = xhr.response?.msg || "上传失败";
            addToast({
              title: msg,
              color: "danger",
              variant: "flat"
            });
            reject(new Error(msg));
          }
        };

        xhr.onerror = () => reject(new Error("网络错误"));
        xhr.onabort = () => reject(new DOMException("Aborted", "AbortError"));

        const abortHandler = () => xhr.abort();

        controller.signal.addEventListener("abort", abortHandler, {
          once: true,
        });

        xhr.send(formData);
      }),
    [updateItem],
  );

  // 大文件分片上传，按已完成分片比例更新进度
  const uploadChunked = useCallback(
    async (item: UploadItem, controller: AbortController) => {
      const totalChunks = Math.ceil(item.file.size / CHUNK_SIZE);
      let uploadedChunks = await fetchUploadedChunks(
        item.uploadId,
        controller.signal,
      );
      let finished = uploadedChunks.size;

      const updateProgress = () => {
        const percent = Math.min(
          99,
          Math.round((finished / totalChunks) * 100),
        );

        updateItem(item.id, () => ({ progress: percent, status: "uploading" }));
      };

      updateProgress();

      for (let index = 0; index < totalChunks; index++) {
        if (controller.signal.aborted) {
          throw new DOMException("Aborted", "AbortError");
        }

        if (uploadedChunks.has(index)) {
          finished += 1;
          updateProgress();
          continue;
        }

        const start = index * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, item.file.size);
        const chunk = item.file.slice(start, end);

        const formData = new FormData();

        formData.append("file", chunk);
        formData.append("uploadId", item.uploadId);
        formData.append("fileName", item.file.name);
        formData.append("fileSize", `${item.file.size}`);
        formData.append("chunkIndex", `${index}`);
        formData.append("totalChunks", `${totalChunks}`);
        formData.append("chunkSize", `${CHUNK_SIZE}`);

        const res = await fetch("/api/upload", {
          method: "POST",
          body: formData,
          signal: controller.signal,
        });

        const json = await res.json().catch(() => null);
        const data = (json?.data ?? {}) as
          | UploadChunkResponse
          | UploadCompletedResponse;

        if (!res.ok) {
          const msg = json?.msg || "分片上传失败";
          addToast({
            title: msg,
            color: "danger",
            variant: "flat"
          });
          throw new Error(json?.msg || "分片上传失败");
        }

        finished += 1;
        updateProgress();

        // 分片接口：如果返回包含分享信息则说明已合并完成
        if ("shareCode" in data && data.shareCode) {
          const payload = data as UploadCompletedResponse;

          updateItem(item.id, () => ({
            progress: 100,
            status: "success",
            shareCode: payload.shareCode,
            resourceId: payload.resourceId,
          }));

          return;
        }

        // 仅分片确认，继续下一片
        if ("merged" in data && data.merged && finished >= totalChunks) {
          // 分片层面已全部完成，但未拿到分享信息，后续由结尾逻辑兜底
          break;
        }
      }

      updateItem(item.id, () => ({
        progress: 100,
        status: "success",
      }));
    },
    [fetchUploadedChunks, updateItem],
  );

  const startUpload = useCallback(
    async (item: UploadItem) => {
      const controller = new AbortController();

      updateItem(item.id, () => ({ status: "uploading", controller }));

      try {
        if (item.file.size > CHUNK_THRESHOLD) {
          await uploadChunked(item, controller);
        } else {
          await uploadSingle(item, controller);
        }
        updateItem(item.id, () => ({ controller: undefined }));
      } catch (error) {
        if ((error as Error).name === "AbortError") {
          removeItem(item.id);

          return;
        }
        updateItem(item.id, () => ({
          status: "error",
          controller: undefined,
          error: (error as Error).message || "上传失败",
        }));
      }
    },
    [removeItem, updateItem, uploadChunked, uploadSingle],
  );

  const handleCancel = useCallback(
    (id: string) => {
      const target = itemsRef.current.find((item) => item.id === id);

      target?.controller?.abort();
      removeItem(id);
    },
    [removeItem],
  );

  const copyText = useCallback(async (text: string) => {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);

        return true;
      }
    } catch {
      /* fallback */
    }
    try {
      const textarea = document.createElement("textarea");

      textarea.value = text;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);

      return true;
    } catch {
      return false;
    }
  }, []);

  const buildShareUrl = useCallback((item: UploadItem) => {
    if (item.shareCode) {
      return new URL(`/s/${item.shareCode}`, window.location.origin).toString();
    }

    return "";
  }, []);

  const handleCopyLink = useCallback(
    async (item: UploadItem) => {
      if (item.status !== "success") return;
      const link = buildShareUrl(item);

      if (!link) return;
      const ok = await copyText(link);
      if (ok) {
        addToast({
          title: "已复制链接",
          description: link,
          color: "success",
          variant: "flat",
        });
      } else {
        addToast({
          title: "复制失败",
          description: "请手动复制链接",
          color: "danger",
          variant: "flat",
        });
      }

      setCopiedId(ok ? item.id : null);
      if (ok) {
        setTimeout(() => {
          setCopiedId((prev) => (prev === item.id ? null : prev));
        }, 1500);
      }
    },
    [buildShareUrl, copyText],
  );

  const handleOpenLink = useCallback(
    (item: UploadItem) => {
      if (item.status !== "success") return;
      const link = buildShareUrl(item);

      if (!link) return;
      window.open(link, "_blank", "noopener,noreferrer");
    },
    [buildShareUrl],
  );

  const handleRetry = useCallback(
    (item: UploadItem) => {
      const newUploadId = createUploadId(item.file);
      const resetState: Partial<UploadItem> = {
        status: "idle",
        progress: 0,
        error: undefined,
        controller: undefined,
        shareCode: undefined,
        resourceId: undefined,
        uploadId: newUploadId,
      };

      updateItem(item.id, () => resetState);
      startUpload({ ...item, ...resetState } as UploadItem);
    },
    [startUpload, updateItem],
  );

  const onCardKeyDown = useCallback(
    (event: KeyboardEvent<HTMLDivElement>, item: UploadItem) => {
      if (item.status !== "success" || !item.shareCode) return;
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        handleCopyLink(item);
      }
    },
    [handleCopyLink],
  );

  // 检测空闲队列，自动启动上传
  useEffect(() => {
    items.forEach((item) => {
      if (item.status === "idle") {
        startUpload(item);
      }
    });
  }, [items, startUpload]);

  return (
    <div
      className="w-full max-w-3xl overflow-hidden"
      style={{
        height: "var(--upload-height)",
      }}
    >
      <div
        className={clsx(
          "flex h-full w-full p-4 md:p-8 flex-col rounded-3xl border backdrop-blur",
          "border-default-200/70 bg-white/40 cursor-pointer",
          "dark:border-default-100/10 dark:bg-default-700/10",
        )}
        style={{ maxHeight: "100%" }}
      >
        <input
          ref={inputRef}
          multiple
          accept="*/*"
          className="hidden"
          type="file"
          onChange={onInputChange}
        />

        {items.length === 0 ? (
          <div
            className={clsx(
              "group flex flex-col items-center justify-center gap-4 rounded-2xl border-2 border-dashed px-8 py-12 text-center transition-all",
              "bg-default-50/60 dark:bg-default-50/5",
              isDragging
                ? "border-primary/70 bg-primary/5 shadow-lg"
                : "border-default-200/80 dark:border-default-500/30 hover:border-primary/60 hover:bg-foreground/5",
            )}
            role="button"
            style={{
              height: "var(--upload-height)",
            }}
            tabIndex={0}
            onClick={onBrowseClick}
            onDragLeave={onDragLeave}
            onDragOver={onDragOver}
            onDrop={onDrop}
            onKeyDown={() => { }}
          >
            <span className="flex h-14 w-14 items-center justify-center rounded-full bg-primary/10 text-primary">
              <CloudArrowUpIcon className="h-7 w-7" />
            </span>
            <div className="space-y-2">
              <p className="text-xl font-semibold text-default-800 dark:text-default-700">
                将文件拖放到这里
              </p>
              <p className="text-sm text-default-500">或</p>
            </div>
            <Button
              className="min-w-[200px]"
              color="primary"
              radius="sm"
              size="md"
              onPress={onBrowseClick}
            >
              选择文件
            </Button>
            <p className="text-xs text-default-500">
              支持图片、视频、音频及其他文件，单文件上限 1GB，可拖拽、点击或粘贴 (Ctrl+V)
            </p>
          </div>
        ) : (
          <div
            className="h-full"
            style={{}}
          >
            <div className="h-[30px] pb-[10px] text-sm font-medium text-default-700 dark:text-default-400">
              已选择的文件
            </div>
            <div
              className={clsx(
                "grid grid-cols-2 sm:grid-cols-4 gap-4 content-start overflow-y-auto upload-scrollbar w-full",
                isDragging
                  ? "ring-1 ring-primary/50 ring-offset-2 ring-offset-white/70 dark:ring-offset-default-50/10"
                  : "",
              )}
              role="button"
              style={{
                height: "calc(100% - 30px)",
              }}
              tabIndex={0}
              onDragLeave={onDragLeave}
              onDragOver={onDragOver}
              onDrop={onDrop}
            >

              {[UploadGridItem, ...items].map((item) => {
                if (item.id === "upload") {
                  return (<div 
                    className={clsx(
                      "sticky top-0 z-100 p-3 rounded-2xl flex items-center justify-center",
                      "rounded-2xl border border-dashed border-default-200/80 bg-white/50 text-default-600",
                      "shadow-sm backdrop-blur transition hover:border-primary/60 hover:text-primary",
                      "dark:border-default-100/30 dark:bg-default-300/90"
                    )}
                    role="button"
                    tabIndex={0}
                    onClick={onBrowseClick}
                    onKeyDown={onUploadKeyDown}
                  >
                    <div
                      className="flex cursor-pointer flex-col items-center justify-center gap-3"
                    >
                      <span className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10 text-primary">
                        <CloudArrowUpIcon className="h-6 w-6" />
                      </span>
                      <span className="text-sm font-medium">继续上传</span>
                      <span className="text-[11px] text-default-500">
                        支持拖拽/点击
                      </span>
                    </div>
                  </div>)
                }

                const isCopyable =
                  item.status === "success" && Boolean(item.shareCode);

                return (
                  <div
                    key={item.id}
                    className={clsx(
                      "relative p-3 rounded-2xl",
                      "border border-default-200/60 bg-white/90 text-default-700 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md",
                      "dark:border-default-400/20 dark:bg-default-500/10",
                    )}
                  >
                    <div
                      className={clsx(
                        "flex aspect-square flex-col overflow-hidden",
                        isCopyable ? "cursor-pointer" : "",
                      )}
                      role={isCopyable ? "button" : undefined}
                      tabIndex={isCopyable ? 0 : -1}
                      title={item.file.name}
                      onClick={() => handleCopyLink(item)}
                      onKeyDown={(e) => onCardKeyDown(e, item)}
                    >
                      <div className="relative z-10 flex-1 overflow-hidden rounded-xl bg-default-100/80 dark:bg-default-50/20">
                        {item.type === "image" && item.previewUrl ? (
                          <Image
                            removeWrapper
                            alt={item.file.name}
                            className="h-full w-full object-cover"
                            src={item.previewUrl}
                          />
                        ) : (
                          <span className="absolute inset-0 flex items-center justify-center text-default-500">
                            {item.type === "video" && (
                              <VideoIcon className="h-10 w-10" />
                            )}
                            {item.type === "audio" && (
                              <AudioIcon className="h-10 w-10" />
                            )}
                            {item.type === "other" && (
                              <FileIcon className="h-10 w-10" />
                            )}
                          </span>
                        )}

                        {/* uploading */}
                        <div className="absolute z-11 pointer-events-none inset-0">
                          <div
                            className="absolute bottom-0 left-0 right-0 bg-sky-300/40 transition-all duration-300"
                            style={{
                              height: `${item.progress}%`,
                              opacity: item.status === "uploading" ? 1 : 0,
                            }}
                          />
                          {item.status === "uploading" && (
                            <div
                              className="absolute inset-0 flex items-end justify-center pb-2 text-xs font-semibold 
                            text-black dark:text-default-600 drop-shadow"
                            >
                              {item.progress}%
                            </div>
                          )}
                        </div>
                      </div>

                      {/* right top buttons */}
                      {item.status === "uploading" && (
                        <button
                          aria-label="取消上传"
                          className={clsx(
                            "absolute right-2 top-2 z-11 flex h-6 w-6 items-center justify-center rounded-full",
                            "shadow-sm transition bg-warning-100 text-warning-600 hover:bg-warning-200",
                          )}
                          onClick={(e) => {
                            e.stopPropagation();
                            handleCancel(item.id);
                          }}
                        >
                          <Icon icon="material-symbols:cancel-outline-rounded" width="16" height="16" />
                        </button>
                      )}
                      {item.status === "success" && (
                        <button
                          aria-label="打开链接"
                          className={clsx(
                            "absolute right-2 top-2 z-11 flex h-6 w-6 items-center justify-center rounded-full",
                            "shadow-sm transition bg-success-100 text-success-600 hover:bg-success-200",
                          )}
                          onClick={(e) => {
                            e.stopPropagation
                            handleOpenLink(item);
                          }}
                        >
                          <Icon icon="qlementine-icons:success-12" width="16" height="16" />
                        </button>
                      )}
                      {item.status === "error" && (
                        <button
                          aria-label="重新上传"
                          className={clsx(
                            "absolute right-2 top-2 z-11 flex h-6 w-6 items-center justify-center rounded-full",
                            "shadow-sm transition bg-danger-100 text-danger-600 hover:bg-danger-200",
                          )}
                          onClick={(e) => {
                            e.stopPropagation();
                            handleRetry(item);
                          }}
                        >
                          <RetryIcon className="h-4 w-4" />
                        </button>
                      )}

                      <p className="mt-2 w-full truncate text-xs text-default-600 dark:text-default-600">
                        {item.file.name}
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
