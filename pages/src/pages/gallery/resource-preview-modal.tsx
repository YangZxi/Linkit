import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Accordion, AccordionItem, Button, Chip, Modal, ModalBody, ModalContent,
  ModalFooter, ModalHeader, Progress, Spinner, addToast,
} from "@heroui/react";
import { Icon } from "@iconify/react";

import type { GalleryItem, ManagedShare } from "@/types/api";
import api from "@/lib/api";
import { copyText } from "@/lib/utils";
import { inferMediaType } from "@/lib/file";
import PreviewCore from "@/components/preview-core";
import ShareFormModal from "./share-form-modal";
import {
  formatShareTime, getResourceShares, isShareExpired, latestPublicShare,
  replaceResourceFile, resourceFileUrl, shareCopyText, shareUrl, sortShares,
} from "./gallery-api";

type Props = {
  resource: GalleryItem;
  onClose: () => void;
  onReplaced: (resource: GalleryItem) => void;
};

export default function ResourcePreviewModal({ resource, onClose, onReplaced }: Props) {
  const fileInput = useRef<HTMLInputElement>(null);
  const [replacing, setReplacing] = useState(false);
  const [progress, setProgress] = useState(0);
  const [shares, setShares] = useState<ManagedShare[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [revision, setRevision] = useState(0);
  const [form, setForm] = useState<{ share?: ManagedShare } | null>(null);
  const [deleting, setDeleting] = useState<ManagedShare | null>(null);
  const [deleteSubmitting, setDeleteSubmitting] = useState(false);
  const [copying, setCopying] = useState(false);
  const [now, setNow] = useState(Date.now());
  const [wide, setWide] = useState(() => window.matchMedia("(min-width: 768px)").matches);

  useEffect(() => {
    const media = window.matchMedia("(min-width: 768px)");
    const update = () => setWide(media.matches);
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setLoadError(false);
    getResourceShares(resource.id).then((items) => {
      if (active) setShares(items);
    }).catch(() => {
      if (active) setLoadError(true);
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, [resource.id, revision]);

  const sortedShares = useMemo(() => sortShares(shares, now), [shares, now]);
  const refreshShares = useCallback(() => setRevision((value) => value + 1), []);

  const copy = async (text: string) => {
    const copied = await copyText(text);
    addToast({ title: copied ? "已复制" : "复制失败，请手动复制", color: copied ? "success" : "danger" });
  };

  const copyRawLink = async () => {
    if (copying) return;
    setCopying(true);
    try {
      const items = await getResourceShares(resource.id);
      setShares(items);
      const share = latestPublicShare(items);
      if (share) {
        await copy(shareUrl(share.code, true));
      } else {
        setForm({});
        addToast({ title: "暂无可用的无密码分享，请先创建", color: "warning" });
      }
    } catch (error) {
      if (error instanceof Error) addToast({ title: error.message, color: "danger" });
    } finally {
      setCopying(false);
    }
  };

  const replace = async (file: File) => {
    if (replacing) return;
    if (file.size > 1024 * 1024 * 1024) {
      addToast({ title: "文件大小不能超过 1GB", color: "danger" });
      return;
    }
    setReplacing(true);
    setProgress(0);
    try {
      const result = await replaceResourceFile(resource, file, setProgress);
      onReplaced(result.resource);
      addToast({ title: result.warning || "文件替换成功", color: result.warning ? "warning" : "success" });
    } catch (error) {
      addToast({ title: (error as Error).message || "文件替换失败", color: "danger" });
    } finally {
      setReplacing(false);
    }
  };

  const deleteShare = async () => {
    if (!deleting || deleteSubmitting) return;
    setDeleteSubmitting(true);
    try {
      await api.post(`/gallery/${resource.id}/shares/${deleting.id}/delete`);
      setShares((items) => items.filter((item) => item.id !== deleting.id));
      setDeleting(null);
      refreshShares();
      addToast({ title: "分享已删除", color: "success" });
    } catch (error) {
      if (error instanceof Error) addToast({ title: error.message, color: "danger" });
    } finally {
      setDeleteSubmitting(false);
    }
  };

  const shareList = (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex shrink-0 items-center justify-between gap-2">
        <span className="text-sm font-semibold">全部分享{!loading && ` (${shares.length})`}</span>
        <Button isIconOnly size="sm" variant="light" aria-label="刷新分享列表"
          isLoading={loading} onPress={refreshShares}><Icon icon="mdi:refresh" width={18} /></Button>
      </div>
      <div className="min-h-0 flex-1 space-y-2 overflow-y-auto pr-1">
        {loading ? <div className="flex justify-center py-6"><Spinner size="sm" /></div> :
          loadError ? <div className="space-y-2 py-4 text-center text-sm text-danger">
            <p>分享列表加载失败</p><Button size="sm" variant="flat" onPress={refreshShares}>重试</Button>
          </div> : sortedShares.length === 0 ?
            <p className="py-6 text-center text-sm text-default-500">暂无分享，创建后即可发送链接</p> :
            sortedShares.map((share) => {
              const expired = isShareExpired(share, now);
              return (
                <div key={share.id} className={`space-y-2 rounded-xl border border-default-200 p-3 ${expired ? "opacity-60" : ""}`}>
                  <div className="flex items-start justify-between gap-2">
                    <span className="min-w-0 break-all font-mono text-sm font-semibold">{share.code}</span>
                    <Chip className="shrink-0" size="sm" variant="flat" color={expired ? "default" : "success"}>
                      {expired ? "已过期" : "有效"}
                    </Chip>
                  </div>
                  <div className="space-y-1 break-all text-xs text-default-500">
                    <p>{share.password ? `密码：${share.password}` : "无需密码"}</p>
                    <p>{share.expireTime ? `到期：${formatShareTime(share.expireTime)}` : "永不过期"}</p>
                    <p>创建：{formatShareTime(share.createdAt)}</p>
                    <p>访问 {share.viewCount} 次{share.relay ? " · 已开启加速" : ""}</p>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button size="sm" variant="flat" className="min-w-0 flex-1"
                      onPress={() => void copy(shareCopyText(share))}>复制</Button>
                    <Button isIconOnly size="sm" variant="light" aria-label={`编辑分享 ${share.code}`}
                      onPress={() => setForm({ share })}><Icon icon="mdi:pencil-outline" width={18} /></Button>
                    <Button isIconOnly size="sm" variant="light" color="danger" aria-label={`删除分享 ${share.code}`}
                      onPress={() => setDeleting(share)}><Icon icon="mdi:trash-can-outline" width={18} /></Button>
                  </div>
                </div>
              );
            })}
      </div>
    </div>
  );

  return (
    <>
      <Modal isOpen isDismissable={false} isKeyboardDismissDisabled={replacing} hideCloseButton={replacing}
        placement="center" size="5xl" scrollBehavior="inside" onClose={onClose}>
        <ModalContent className="max-h-[92dvh]">
          <ModalHeader className="flex items-center justify-between gap-3 pr-12">
            <div className="min-w-0">
              <p className="text-lg font-semibold">预览</p>
              <p className="truncate text-sm font-normal text-default-500">{resource.filename}</p>
            </div>
            <Button className="shrink-0" size="sm" variant="flat" color="secondary" isLoading={replacing}
              onPress={() => fileInput.current?.click()}>
              {replacing ? `上传中 ${progress}%` : "替换文件"}
            </Button>
            <input ref={fileInput} aria-label="选择替换文件" className="hidden" type="file" disabled={replacing}
              onChange={(event) => {
                const file = event.target.files?.[0];
                event.target.value = "";
                if (file) void replace(file);
              }} />
          </ModalHeader>
          <ModalBody className="gap-3">
            {replacing && <Progress aria-label="文件上传进度" value={progress} size="sm" />}
            <div className="flex min-h-0 flex-col gap-4 md:h-[min(470px,55dvh)] md:flex-row">
              <div className="flex h-[min(360px,45dvh)] min-h-[200px] min-w-0 flex-1 overflow-hidden md:h-full [&_img]:max-h-full">
                <PreviewCore key={`${resource.id}:${resource.contentVersion}`} className="h-full w-full min-w-0"
                  filename={resource.filename} rawUrl={resourceFileUrl(resource)} type={inferMediaType(resource.type)} />
              </div>
              {wide && <aside aria-label="分享管理" className="flex h-full w-[240px] shrink-0 rounded-2xl border border-default-200 p-3">
                {shareList}
              </aside>}
            </div>
            {!wide && <Accordion key={resource.id} className="px-0" variant="bordered">
              <AccordionItem key="shares" aria-label="分享管理" title="分享管理" subtitle={`${shares.length} 条分享`}>
                <div className="flex h-[300px] max-h-[40dvh] min-h-0 flex-col pb-2">{shareList}</div>
              </AccordionItem>
            </Accordion>}
          </ModalBody>
          <ModalFooter className="flex-wrap gap-2">
            <Button size="sm" color="secondary" variant="bordered" isLoading={copying} onPress={() => void copyRawLink()}>
              获取原始链接
            </Button>
            <Button size="sm" color="primary" variant="flat" onPress={() => setForm({})}>创建分享</Button>
            <Button size="sm" as="a" color="primary" download={resource.filename} href={resourceFileUrl(resource)} variant="flat">下载</Button>
            <Button size="sm" variant="flat" isDisabled={replacing} onPress={onClose}>关闭</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
      {form && <ShareFormModal key={form.share?.id || "new"} resource={resource} share={form.share}
        onClose={() => setForm(null)} onSaved={refreshShares} />}
      {deleting && <Modal isOpen isDismissable={!deleteSubmitting} isKeyboardDismissDisabled={deleteSubmitting}
        placement="center" onClose={() => !deleteSubmitting && setDeleting(null)}>
        <ModalContent>
          <ModalHeader>删除分享</ModalHeader>
          <ModalBody><p>删除分享 {deleting.code} 后，该链接将立即失效，文件仍然保留。</p></ModalBody>
          <ModalFooter>
            <Button variant="flat" isDisabled={deleteSubmitting} onPress={() => setDeleting(null)}>取消</Button>
            <Button color="danger" isLoading={deleteSubmitting} onPress={() => void deleteShare()}>确认删除</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>}
    </>
  );
}
