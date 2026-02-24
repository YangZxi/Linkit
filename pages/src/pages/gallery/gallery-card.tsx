import type { GalleryItem } from "@/types/api";

import { Button, Image } from "@heroui/react";
import { Icon } from "@iconify/react";

import { inferMediaType, type MediaType, TypeLabel } from "@/lib/file";

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

type GalleryCardProps = {
  item: GalleryItem;
  origin: string;
  onCopyLink: (url: string) => void;
  onDelete: (item: GalleryItem) => void;
  onShare: () => void;
  deleting: boolean;
};

export default function GalleryCard({
  item,
  origin,
  onCopyLink,
  onDelete,
  onShare,
  deleting,
}: GalleryCardProps) {
  const type = inferMediaType(item.type || "");
  const rawUrl = item.shareCode ? `/r/${item.shareCode}` : "";
  const shareUrl =
    item.shareCode && origin ? `${origin}/s/${item.shareCode}` : "";
  const storageLabel = item.storage === "local" ? "本地" : "S3";

  const cover = (() => {
    if (!rawUrl) {
      return <div className="h-full w-full flex justify-center items-center text-sm text-default-500">
        资源不可用
      </div>;
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
            {storageLabel} {item.shareCode}
          </span>
        ) : (
          <span className="absolute bottom-3 right-3 rounded-full bg-danger-500/80 px-3 py-1 text-[11px] text-white backdrop-blur">
            暂无短链（{storageLabel}）
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
        <div className="text-[11px] text-default-500" />
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
