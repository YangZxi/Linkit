import type {
  GalleryDeleteResponse,
  GalleryItem,
  GalleryResponse,
  CreateShareResponse,
} from "@/types/api";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Button,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
  Spinner,
  Image,
  addToast,
  Input,
  Alert,
  Snippet,
} from "@heroui/react";
import clsx from "clsx";
import { Icon } from "@iconify/react";

import PreviewCore from "./preview-core";

import api, { ApiResponse } from "@/lib/api";
import { inferMediaType, MediaType, TypeLabel } from "@/lib/file";
import { copyText } from "@/lib/utils";
import XModal from "./modal";

const PAGE_SIZE = 10;
const SHARE_PASSWORD_MIN = 4;
const SHARE_PASSWORD_MAX = 32;

function formatDateText(value: string) {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) return "-";

  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function TypeMark({ type }: { type: MediaType }) {
  return (
    <span className="rounded-full bg-black/60 px-3 py-1 text-[11px] font-medium text-white backdrop-blur dark:bg-white/15">
      {TypeLabel[type]}
    </span>
  );
}

function GalleryCard({
  item,
  origin,
  onCopyLink,
  onDelete,
  onShare,
  deleting,
}: {
  item: GalleryItem;
  origin: string;
  onCopyLink: (url: string) => void;
  onDelete: (item: GalleryItem) => void;
  onShare: () => void;
  deleting: boolean;
}) {
  const type = inferMediaType(item.type || "");
  const rawUrl = item.shareCode ? `/r/${item.shareCode}` : "";
  const shareUrl =
    item.shareCode && origin ? `${origin}/s/${item.shareCode}` : "";

  const cover = (() => {
    if (!rawUrl) {
      return <div className="h-full w-full flex justify-center items-center text-sm text-default-500">
        资源不可用
      </div>
    }
    if (type === "image" && rawUrl) {
      return (
        <Image
          removeWrapper
          alt={item.filename}
          className="z-auto h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
          src={rawUrl}
        />
      );
    }
    if (type === "video") {
      return (
        <div className="flex h-full w-full flex-col items-center justify-center gap-2 bg-gradient-to-br from-default-100 via-default-200 to-default-100 text-default-600 dark:from-default-50/10 dark:via-default-100/10 dark:to-default-50/10">
          <svg
            aria-hidden="true"
            className="h-8 w-8"
            fill="none"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d="M15 12l-5 3V9l5 3z" fill="currentColor" />
            <rect
              height="14"
              rx="2.5"
              stroke="currentColor"
              strokeWidth="1.5"
              width="14"
              x="4"
              y="5"
            />
          </svg>
          <p className="text-xs text-default-500">点击复制链接后播放</p>
        </div>
      );
    }
    if (type === "audio") {
      return (
        <div className="flex h-full w-full flex-col items-center justify-center gap-2 bg-gradient-to-br from-default-100 via-default-200 to-default-100 text-default-600 dark:from-default-50/10 dark:via-default-100/10 dark:to-default-50/10">
          <svg
            aria-hidden="true"
            className="h-7 w-7"
            fill="none"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M8 15.5A2.5 2.5 0 1010.5 18V7.5L16 6v8.5"
              stroke="currentColor"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="1.5"
            />
            <circle
              cx="16"
              cy="16.5"
              r="2.5"
              stroke="currentColor"
              strokeWidth="1.5"
            />
          </svg>
          <p className="text-xs text-default-500">音频资源</p>
        </div>
      );
    }

    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-2 bg-gradient-to-br from-default-100 via-default-200 to-default-100 text-default-700 dark:from-default-50/10 dark:via-default-100/10 dark:to-default-50/10">
        <svg
          aria-hidden="true"
          className="h-8 w-8"
          fill="none"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            d="M9 4h4.5L18 8.5V20a1 1 0 01-1 1H9a1 1 0 01-1-1V5a1 1 0 011-1z"
            stroke="currentColor"
            strokeLinejoin="round"
            strokeWidth="1.5"
          />
          <path
            d="M13 4v4.5a1 1 0 001 1H18"
            stroke="currentColor"
            strokeLinejoin="round"
            strokeWidth="1.5"
          />
        </svg>
        <p className="text-xs text-default-500">文件资源</p>
      </div>
    );
  })();

  return (
    <div className="group flex flex-col overflow-hidden rounded-2xl border border-default-200/80 bg-white/80 shadow-sm transition hover:-translate-y-1 hover:shadow-lg dark:border-default-100/20 dark:bg-default-50/10">
      <div className="relative aspect-square overflow-hidden">
        {cover}
        <div className="absolute left-3 top-3">
          <TypeMark type={type} />
        </div>
        {item.shareCode ? (
          <span className="absolute bottom-3 right-3 rounded-full bg-default-900/70 px-3 py-1 text-[11px] text-white backdrop-blur dark:bg-default-100/30">
            短链 {item.shareCode}
          </span>
        ) : (
          <span className="absolute bottom-3 right-3 rounded-full bg-danger-500/80 px-3 py-1 text-[11px] text-white backdrop-blur">
            暂无短链
          </span>
        )}
      </div>
      <div className="flex flex-col gap-1 px-3 pb-3 pt-2">
        <p className="truncate text-sm font-semibold text-default-900 dark:text-default-700">
          {item.filename}
        </p>
        <p className="text-xs text-default-500">
          上传于 {formatDateText(item.createdAt)}
        </p>
      </div>
      <div className="flex items-center justify-between px-3 pb-3">
        <div className="text-[11px] text-default-500">

        </div>
        <div className="flex items-center gap-2">
          <Button
            isIconOnly
            aria-label="删除资源"
            color="danger"
            isDisabled={deleting}
            size="sm"
            variant="flat"
            onClick={(event) => event.stopPropagation()}
            onPress={() => onDelete(item)}
          >
            <Icon className="text-lg" icon="ic:outline-delete" />
          </Button>
          <Button
            isIconOnly
            aria-label="加密分享"
            color="secondary"
            size="sm"
            variant="flat"
            onClick={(event) => event.stopPropagation()}
            onPress={() => onShare()}
          >
            <Icon className="text-lg" icon="mdi:send-lock-outline" />
          </Button>
          <Button
            color="primary"
            isDisabled={!shareUrl}
            size="sm"
            variant="flat"
            onClick={(event) => event.stopPropagation()}
            onPress={() => shareUrl && onCopyLink(shareUrl)}
          >
            复制链接
          </Button>
        </div>
      </div>
    </div>
  );
}

export default function GalleryGrid() {
  const [items, setItems] = useState<GalleryItem[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [origin, setOrigin] = useState("");
  const [preview, setPreview] = useState<GalleryItem | null>(null);
  const [share, setShare] = useState<GalleryItem | null>(null);
  const [sharePassword, setSharePassword] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<GalleryItem | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [shareSubmitting, setShareSubmitting] = useState(false);
  const [shareResult, setShareResult] = useState<{
    url: string;
    code: string;
    password: string;
  } | null>(null);

  useEffect(() => {
    if (typeof window !== "undefined") {
      setOrigin(window.location.origin);
    }
  }, []);

  useEffect(() => {
    if (share) {
      setSharePassword("");
      setShareResult(null);
    }
  }, [share]);

  const totalPages = useMemo(() => {
    if (total <= 0) return 1;

    return Math.max(1, Math.ceil(total / PAGE_SIZE));
  }, [total]);

  const fetchData = useCallback(
    async (targetPage: number) => {
      const nextPage = Math.max(1, targetPage);

      setLoading(true);
      setError(null);
      try {
        const res = await api.get<GalleryResponse>(
          `/gallery?page=${nextPage}&size=${PAGE_SIZE}`,
        );
        const maxPage =
          res.total > 0 ? Math.max(1, Math.ceil(res.total / PAGE_SIZE)) : 1;
        const safePage = Math.min(res.page, maxPage);

        if (safePage !== page) {
          setPage(safePage);
        }
        setItems(res.data);
        setTotal(res.total);
      } catch (err) {
        console.log(err)
        const message = (err as ApiResponse<unknown>).msg;
        setError(message);
        setItems([]);
        setTotal(0);
      } finally {
        setLoading(false);
      }
    },
    [page],
  );

  useEffect(() => {
    fetchData(page);
  }, [fetchData, page]);

  const handleCopy = useCallback(async (url: string) => {
    if (!url) return;

    const ok = await copyText(url);
    if (ok) {
      addToast({
        title: "已复制链接",
        description: url,
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
  }, []);

  const handleDelete = useCallback(
    async (item: GalleryItem) => {
      if (deletingId === item.id) return false;

      setDeletingId(item.id);
      try {
        await api.post<GalleryDeleteResponse>("/gallery/delete", {
          id: item.id,
        });
        addToast({
          title: "删除成功",
          description: item.filename,
          color: "success",
          variant: "flat",
        });
        if (preview?.id === item.id) {
          setPreview(null);
        }
        await fetchData(page);
        return true;
      } catch (err: any) {
        return false;
      } finally {
        setDeletingId(null);
      }
    },
    [deletingId, fetchData, page, preview],
  );

  const confirmDelete = useCallback(async () => {
    if (!deleteTarget) return;

    const ok = await handleDelete(deleteTarget);

    if (ok) {
      setDeleteTarget(null);
    }
  }, [deleteTarget, handleDelete]);

  const gotoPrev = useCallback(() => {
    setPage((prev) => Math.max(1, prev - 1));
  }, []);

  const gotoNext = useCallback(() => {
    setPage((prev) => prev + 1);
  }, []);

  const submitShare = useCallback(async () => {
    if (!share || shareSubmitting) return;
    if (shareResult) {
      const copyContent = `分享链接：${shareResult.url}\n密码：${shareResult.password}`;
      const ok = await copyText(copyContent);
      addToast({
        title: ok ? "已复制完整分享信息" : "复制失败",
        color: ok ? "success" : "danger",
        variant: "flat",
      });
      return;
    }
    const trimmedPassword = sharePassword.trim();
    const length = trimmedPassword.length;
    if (length < SHARE_PASSWORD_MIN || length > SHARE_PASSWORD_MAX) {
      addToast({
        title: "分享密码长度不合法",
        description: `请设置 ${SHARE_PASSWORD_MIN}-${SHARE_PASSWORD_MAX} 位密码`,
        color: "warning",
        variant: "flat",
      });
      return;
    }
    setShareSubmitting(true);
    try {
      const res = await api.post<CreateShareResponse>("/share", {
        resourceId: share.id,
        password: trimmedPassword,
      });
      const shareUrl = origin
        ? `${origin}/s/${res.code}`
        : `/s/${res.code}`;
      addToast({
        title: "私密分享创建成功",
        description: "可复制分享信息发送给对方",
        color: "success",
        variant: "flat",
      });
      setSharePassword(trimmedPassword);
      setShareResult({
        url: shareUrl,
        code: res.code,
        password: trimmedPassword,
      });
      await fetchData(page);
    } finally {
      setShareSubmitting(false);
    }
  }, [fetchData, origin, page, share, shareResult, sharePassword, shareSubmitting]);

  const isDeleting = Boolean(deleteTarget && deletingId === deleteTarget.id);

  const renderContent = () => {
    if (loading && items.length === 0) {
      return (
        <div className="flex min-h-[320px] items-center justify-center rounded-2xl border border-dashed border-default-200/80 bg-default-50/40 dark:border-default-100/30 dark:bg-default-50/5">
          <div className="flex items-center gap-3 text-default-500">
            <Spinner color="primary" />
            <span>加载中...</span>
          </div>
        </div>
      );
    }

    if (error) {
      return (
        <div className="flex min-h-[320px] flex-col items-center justify-center gap-3 rounded-2xl border border-danger-100 bg-danger-50/50 text-danger-500 dark:border-danger-200/20 dark:bg-danger-100/5">
          <p>加载失败：{error}</p>
          <Button
            color="primary"
            variant="flat"
            onPress={() => fetchData(page)}
          >
            重试
          </Button>
        </div>
      );
    }

    if (!items.length) {
      return (
        <div className="flex min-h-[320px] flex-col items-center justify-center gap-2 rounded-2xl border border-default-200/80 bg-default-50/40 text-default-500 dark:border-default-100/30 dark:bg-default-50/5">
          <p className="text-sm font-medium">暂无资源</p>
          <p className="text-xs">上传后即可在这里查看你的资源列表</p>
        </div>
      );
    }

    return (
      <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
        {items.map((item) => (
          <div
            key={item.id}
            className="cursor-pointer focus:outline-none"
            role="button"
            tabIndex={0}
            onClick={() => {
              if (!item.shareCode) {
                addToast({
                  title: "无法预览",
                  description: "该资源缺少分享短链，无法打开预览",
                  color: "warning",
                  variant: "flat",
                });

                return;
              }
              setPreview(item);
            }}
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                if (item.shareCode) {
                  setPreview(item);
                } else {
                  addToast({
                    title: "无法预览",
                    description: "该资源缺少分享短链，无法打开预览",
                    color: "warning",
                    variant: "flat",
                  });
                }
              }
            }}
          >
            <GalleryCard
              deleting={deletingId === item.id}
              item={item}
              origin={origin}
              onCopyLink={handleCopy}
              onDelete={(target) => setDeleteTarget(target)}
              onShare={() => setShare(item)}
            />
          </div>
        ))}
      </div>
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div />
        <div className="flex items-center gap-3 text-sm text-default-500">
          <span>共 {total} 个</span>
          <span>
            第 {Math.min(page, totalPages)} / {totalPages} 页
          </span>
          <Button
            isLoading={loading}
            color="primary"
            size="sm"
            variant="flat"
            onPress={() => fetchData(page)}
          >
            刷新
          </Button>
        </div>
      </div>

      {renderContent()}

      <div
        className={clsx(
          "flex items-center justify-center gap-3",
          totalPages <= 1 ? "opacity-50" : "",
        )}
      >
        <Button
          isDisabled={page <= 1 || loading}
          size="sm"
          variant="flat"
          onPress={gotoPrev}
        >
          上一页
        </Button>
        <span className="text-sm text-default-600 dark:text-default-400">
          第 {Math.min(page, totalPages)} / {totalPages} 页
        </span>
        <Button
          isDisabled={page >= totalPages || loading}
          size="sm"
          variant="flat"
          onPress={gotoNext}
        >
          下一页
        </Button>
      </div>

      <Modal
        isDismissable={!isDeleting}
        isOpen={Boolean(deleteTarget)}
        placement="center"
        size="md"
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <ModalContent>
          {(close) =>
            deleteTarget ? (
              <>
                <ModalHeader className="flex flex-col gap-1">
                  <span className="text-lg font-semibold text-default-900 dark:text-default-50">
                    确认删除
                  </span>
                  <span className="text-sm text-default-500">
                    删除后无法恢复，请确认。
                  </span>
                </ModalHeader>
                <ModalBody>
                  <p className="text-sm text-default-600 dark:text-default-400">
                    将删除资源：{deleteTarget.filename}
                  </p>
                </ModalBody>
                <ModalFooter>
                  <Button
                    color="default"
                    isDisabled={isDeleting}
                    variant="flat"
                    onPress={close}
                  >
                    取消
                  </Button>
                  <Button
                    color="danger"
                    isLoading={isDeleting}
                    onPress={confirmDelete}
                  >
                    确认删除
                  </Button>
                </ModalFooter>
              </>
            ) : null
          }
        </ModalContent>
      </Modal>


      {/* preview modal */}
      <XModal
        isDismissable={false}
        isOpen={Boolean(preview)}
        placement="center"
        size="4xl"
        onOpenChange={(open) => !open && setPreview(null)}
        header={<>
          <span className="text-lg font-semibold text-default-900 dark:text-default-50">
            预览
          </span>
          <span className="text-sm text-default-500">
            {preview?.filename}
          </span>
        </>}
        footer={(preview && preview.shareCode) && <>
          <Button
            color="secondary"
            variant="bordered"
            onPress={() =>
              handleCopy(
                origin
                  ? `${origin}/r/${preview.shareCode}`
                  : `/r/${preview.shareCode}`,
              )
            }
          >
            获取原始链接
          </Button>
          <Button
            as="a"
            color="primary"
            download
            href={`/r/${preview.shareCode}`}
            variant="flat"
          >
            下载
          </Button>
          <Button color="default" variant="flat" onPress={() => setPreview(null)}>
            关闭
          </Button>
        </>}
      >
        {preview && preview.shareCode ? (
          <PreviewCore
            className="min-h-[260px] max-h-[470px] w-full"
            filename={preview.filename}
            rawUrl={`/r/${preview.shareCode}`}
            type={inferMediaType(preview.type)}
          />
        ) : (<p className="text-default-500">
          该资源缺少短链信息，请重新生成后再试。
        </p>)}
      </XModal>

      {/* share preview modal */}
      <XModal
        isDismissable={false}
        isOpen={Boolean(share)}
        header={<>
          <span className="text-lg font-semibold">
            创建私密分享
          </span>
          <span className="text-sm text-default-500">
            {share?.filename}
          </span>
        </>}
        submitText={shareResult ? "复制" : "创建分享"}
        onSubmit={submitShare}
        onOpenChange={(open) => !open && setShare(null)}
      >
        <Input
          autoFocus
          isDisabled={shareSubmitting || Boolean(shareResult)}
          label="分享密码"
          maxLength={SHARE_PASSWORD_MAX}
          minLength={SHARE_PASSWORD_MIN}
          type="text"
          value={sharePassword}
          onValueChange={setSharePassword}
        />
        {shareResult && (
          <Alert
            color="success"
            description={null
            }
            variant="flat"
          >
            <div className="space-y-1 text-sm text-default-600">
              <p className="font-bold">创建私密分享成功</p>
              <p className="break-all">
                链接：
                <span className="text-primary underline cursor-pointer"
                  onClick={async (event) => {
                    const ok = await copyText(shareResult.url);
                    addToast({
                      title: ok ? "已复制分享链接" : "复制失败",
                      color: ok ? "success" : "danger",
                      variant: "flat",
                    });
                  }}
                >{shareResult.url}</span>
              </p>
              <p>密码：{shareResult.password}</p>
            </div>
          </Alert>
        )}
        {/* <XCalendar /> */}
      </XModal>
    </div >
  );
}
