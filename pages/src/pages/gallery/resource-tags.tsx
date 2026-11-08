import { useEffect, useRef, useState } from "react";
import { Button, Popover, PopoverContent, PopoverTrigger, Spinner, addToast } from "@heroui/react";
import { Icon } from "@iconify/react";

import type { GalleryTagsResponse } from "@/types/api";
import api, { type ApiResponse } from "@/lib/api";

type Props = {
  resourceId: number;
  tags: string[];
  isDisabled: boolean;
  onChanged: (tags: string[]) => void;
};

export default function ResourceTags({ resourceId, tags, isDisabled, onChanged }: Props) {
  const [open, setOpen] = useState(false);
  const [availableTags, setAvailableTags] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [revision, setRevision] = useState(0);
  const [saving, setSaving] = useState(false);
  const [portalContainer, setPortalContainer] = useState<HTMLDivElement | null>(null);
  const pending = useRef(false);

  useEffect(() => {
    if (!open) return;
    let active = true;
    setLoading(true);
    setLoadError(false);
    api.get<GalleryTagsResponse>("/gallery/tags", { hideToast: true }).then((result) => {
      if (active) setAvailableTags(result.tags);
    }).catch(() => {
      if (active) setLoadError(true);
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, [open, revision]);

  const changeTag = async (tag: string, remove = false) => {
    if (pending.current || isDisabled) return;
    pending.current = true;
    setSaving(true);
    try {
      const result = await api.post<GalleryTagsResponse>(`/gallery/${resourceId}/tags`, { tag, remove }, { hideToast: true });
      onChanged(result.tags);
      if (!remove) setOpen(false);
    } catch (error) {
      const message = (error as ApiResponse<unknown> | null)?.msg || "更新标签失败，请检查网络后重试";
      addToast({ title: message, color: "danger" });
    } finally {
      pending.current = false;
      setSaving(false);
    }
  };

  const choices = availableTags.filter((tag) => !tags.includes(tag));

  return (
    <div ref={setPortalContainer} aria-label="资源标签" className="mt-2 flex flex-wrap items-center gap-1.5 font-normal">
      {tags.map((tag) => (
        <span key={tag} className="relative inline-flex min-h-5 max-w-full items-center rounded-md bg-default-100 py-0.5 pl-1.5 pr-4 text-[11px] leading-4 text-default-600">
          <span className="break-all">{tag}</span>
          <button type="button" aria-label={`移除标签 ${tag}`} disabled={isDisabled || saving}
            className="absolute -right-0.5 -top-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-default-200 text-default-500 hover:bg-default-300 hover:text-default-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-focus disabled:opacity-40"
            onClick={() => void changeTag(tag, true)}>
            <Icon icon="mdi:close" width={10} />
          </button>
        </span>
      ))}
      <Popover isOpen={open} onOpenChange={setOpen} placement="bottom-start" portalContainer={portalContainer ?? undefined}>
        <PopoverTrigger>
          <Button isIconOnly aria-label="添加标签" isDisabled={isDisabled || saving} size="sm" variant="flat"
            className="h-5 min-h-5 w-5 min-w-5 rounded-md">
            <Icon icon="mdi:plus" width={14} />
          </Button>
        </PopoverTrigger>
        <PopoverContent aria-label="添加已有标签" className="max-w-[calc(100vw-2rem)] items-stretch p-3">
          <div className="w-60 max-w-full space-y-2">
            <p className="text-xs font-medium">添加已有标签</p>
            {loading ? <div className="flex justify-center py-3"><Spinner size="sm" aria-label="加载标签" /></div> :
              loadError ? <div className="space-y-2 text-xs text-danger">
                <p>标签加载失败</p>
                <Button size="sm" variant="flat" onPress={() => setRevision((value) => value + 1)}>重试</Button>
              </div> : choices.length === 0 ? <p className="py-2 text-xs text-default-500">暂无可添加的标签</p> :
                <div className="flex max-h-48 flex-wrap gap-1.5 overflow-y-auto py-1">
                  {choices.map((tag) => (
                    <Button key={tag} aria-label={`添加标签 ${tag}`} isDisabled={isDisabled || saving}
                      size="sm" variant="flat" className="h-6 min-w-0 rounded-md px-2 text-xs"
                      onPress={() => void changeTag(tag)}>{tag}</Button>
                  ))}
                </div>}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
