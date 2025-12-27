"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { addToast, Button, Input } from "@heroui/react";
import clsx from "clsx";

import PreviewCore from "./PreviewCore";

import { MediaType } from "@/lib/file";
import { copyText } from "@/lib/utils";

type SharePreviewProps = {
  code: string;
  type: MediaType;
  rawUrl: string;
  filename: string;
};

export default function SharePreview({
  code,
  type,
  rawUrl,
  filename,
}: SharePreviewProps) {

  const copy = useCallback(async (text: string, type: "url" | "raw") => {
    if (!text) return;
    const ok = await copyText(text);
    if (ok) {
      addToast({
        title: "已复制链接",
        description: text,
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

  const fullRawUrl = useMemo(() => {
    return new URL(`${rawUrl}`, origin).toString();
  }, [origin, rawUrl]);

  const fullPreviewUrl = useMemo(() => {
    console.log(fullRawUrl);
    const pwd = new URL(fullRawUrl).searchParams.get("pwd");
    return new URL(`/s/${code}${pwd ? `?pwd=${encodeURIComponent(pwd)}` : ""}`, origin).toString();
  }, [code, origin, fullRawUrl]);

  return (
    <div
      className="p-6 flex h-full flex-1 flex-col gap-6 rounded-3xl border border-default-200/60 bg-white/70 shadow-lg backdrop-blur dark:border-default-100/20 dark:bg-default-50/10"
    >
      <div className="space-y-2">
        <p className="text-lg font-semibold text-default-900 dark:text-default-700 break-all">
          {filename}
        </p>
      </div>
      <PreviewCore
        className="flex min-h-[100px] md:min-h-[360px] w-full"
        // style={{ height: contentHeight }}
        filename={filename}
        rawUrl={fullRawUrl}
        type={type}
      />
      <div className="flex flex-col gap-3">
        <Input
          isReadOnly
          classNames={{
            inputWrapper:
              "border border-default-200/60 bg-white/70 dark:border-default-100/30 dark:bg-default-50/5",
          }}
          label="分享链接"
          labelPlacement="outside"
          value={fullPreviewUrl}
          onClick={(event) => {
            (event.target as HTMLInputElement).select && (event.target as HTMLInputElement)?.select();
          }}
        />
        <div className="flex flex-wrap gap-3">
          <Button
            className={clsx(
              "min-w-[120px]",
            )}
            color="primary"
            variant="flat"
            onPress={() => copy(fullPreviewUrl, "url")}
          >
            复制链接
          </Button>
          <Button
            className={clsx(
              "min-w-[120px]",
            )}
            color="secondary"
            variant="flat"
            onPress={() => copy(fullRawUrl, "raw")}
          >
            复制原始链接
          </Button>
          <Button
            as="a"
            className="min-w-[120px]"
            color="primary"
            download
            href={fullRawUrl}
            variant="bordered"
          >
            下载
          </Button>
        </div>
      </div>
    </div>
  );
}
