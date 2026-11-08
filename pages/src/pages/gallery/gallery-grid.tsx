import type {
  GalleryDeleteResponse,
  GalleryItem,
  GalleryResponse,
  GalleryTagsResponse,
} from "@/types/api";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Button,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
  Spinner,
  addToast,
  CheckboxGroup,
  Select,
  SelectItem,
} from "@heroui/react";
import clsx from "clsx";

import GalleryCard from "./gallery-card";
import ResourcePreviewModal from "./resource-preview-modal";
import ShareFormModal from "./share-form-modal";
import { getResourceShares, latestPublicShare, shareUrl } from "./gallery-api";

import api, { ApiResponse } from "@/lib/api";
import { copyText } from "@/lib/utils";
import BillCheckbox from "@/components/bill-checkbox";

const DEFAULT_PAGE_SIZE = 10;
const PAGE_SIZE_OPTIONS = [10, 20, 30, 50] as const;
type PageSize = (typeof PAGE_SIZE_OPTIONS)[number];

function deleteGalleryItems(ids: number[]) {
  return Promise.all(
    ids.map((id) =>
      api.post<GalleryDeleteResponse>("/gallery/delete", { id }),
    ),
  );
}

export default function GalleryGrid() {
  const [availableTags, setAvailableTags] = useState<string[]>([]);
  const [tagSelected, setTagSelected] = useState<string[]>([]);
  const [items, setItems] = useState<GalleryItem[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<PageSize>(DEFAULT_PAGE_SIZE);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [preview, setPreview] = useState<GalleryItem | null>(null);
  const listRequestId = useRef(0);

  const [share, setShare] = useState<GalleryItem | null>(null);
  const [copyingId, setCopyingId] = useState<number | null>(null);

  const [bathDeleteMode, setBathDeleteMode] = useState(false);
  const [batchDeleteIds, setBatchDeleteIds] = useState<number[]>([]);
  const [deleteConfirmItems, setDeleteConfirmItems] = useState<GalleryItem[]>(
    [],
  );
  const [deleteSubmitting, setDeleteSubmitting] = useState(false);
  useEffect(() => {
    if (!bathDeleteMode) {
      setBatchDeleteIds([]);
      setDeleteConfirmItems([]);
    }
  }, [bathDeleteMode]);

  const totalPages = useMemo(() => {
    if (total <= 0) return 1;

    return Math.max(1, Math.ceil(total / pageSize));
  }, [pageSize, total]);

  const fetchTags = useCallback(async () => {
    try {
      const res = await api.get<GalleryTagsResponse>("/gallery/tags", {
        hideToast: true,
      });
      setAvailableTags(res.tags);
      setTagSelected((current) => {
        const next = current.filter((tag) => res.tags.includes(tag));
        return next.length === current.length ? current : next;
      });
    } catch (err) {
      console.log(err);
      setAvailableTags([]);
    }
  }, []);

  const fetchData = useCallback(
    async (
      targetPage: number,
      selectedTags: string[] = tagSelected,
      selectedPageSize: PageSize = pageSize,
    ) => {
      const requestId = ++listRequestId.current;
      const nextPage = Math.max(1, targetPage);
      const params = new URLSearchParams({
        page: String(nextPage),
        size: String(selectedPageSize),
      });
      if (selectedTags.length > 0) {
        params.set("tags", selectedTags.join(","));
      }

      setLoading(true);
      setError(null);
      try {
        const res = await api.get<GalleryResponse>(
          `/gallery?${params.toString()}`,
          { hideToast: true },
        );
        if (requestId !== listRequestId.current) return;
        const maxPage =
          res.total > 0
            ? Math.max(1, Math.ceil(res.total / selectedPageSize))
            : 1;
        const safePage = Math.min(res.page, maxPage);

        if (safePage !== page) {
          setPage(safePage);
        }
        setItems(res.data);
        setTotal(res.total);
      } catch (err) {
        if (requestId !== listRequestId.current) return;
        console.log(err);
        const message = (err as ApiResponse<unknown> | null)?.msg || "获取资源失败，请稍后重试";
        setError(message);
        setItems([]);
        setTotal(0);
        addToast({ title: message, color: "danger" });
      } finally {
        if (requestId === listRequestId.current) setLoading(false);
      }
    },
    [page, pageSize, tagSelected],
  );

  useEffect(() => {
    fetchTags();
  }, [fetchTags]);

  useEffect(() => {
    fetchData(page, tagSelected, pageSize);
    return () => { listRequestId.current += 1; };
  }, [fetchData, page, pageSize, tagSelected]);

  const handleTagChange = useCallback((values: string[]) => {
    setTagSelected(values);
    setPage(1);
  }, []);

  const handlePageSizeChange = useCallback((key?: string | null) => {
    const nextPageSize = Number(key);

    if (PAGE_SIZE_OPTIONS.includes(nextPageSize as PageSize)) {
      setPageSize(nextPageSize as PageSize);
      setPage(1);
    }
  }, []);

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

  const confirmDelete = useCallback(async () => {
    if (deleteConfirmItems.length === 0 || deleteSubmitting) return;

    const deletingItems = deleteConfirmItems;
    const deletingIds = deletingItems.map((item) => item.id);
    const isBatchDelete = deletingItems.length > 1;

    setDeleteSubmitting(true);
    try {
      await deleteGalleryItems(deletingIds);
      addToast({
        title: isBatchDelete ? "批量删除成功" : "删除成功",
        description: isBatchDelete
          ? `已删除 ${deletingItems.length} 个资源`
          : deletingItems[0]?.filename,
        color: "success",
        variant: "flat",
      });
      setDeleteConfirmItems([]);
      setBatchDeleteIds([]);
      if (isBatchDelete) {
        setBathDeleteMode(false);
      }
      if (preview && deletingIds.includes(preview.id)) {
        setPreview(null);
      }
      await fetchData(page);
    } catch (err) {
      console.log(err);
      addToast({
        title: isBatchDelete ? "批量删除失败" : "删除失败",
        description: isBatchDelete
          ? "部分资源可能已删除，请刷新后确认"
          : "请稍后重试",
        color: "danger",
        variant: "flat",
      });
      await fetchData(page);
    } finally {
      setDeleteSubmitting(false);
    }
  }, [deleteConfirmItems, deleteSubmitting, fetchData, page, preview]);

  const toggleBatchDeleteItem = useCallback((item: GalleryItem) => {
    setBatchDeleteIds((prev) => {
      if (prev.includes(item.id)) {
        return prev.filter((id) => id !== item.id);
      }

      return [...prev, item.id];
    });
  }, []);

  const selectedDeleteCount = batchDeleteIds.length;
  const deleteConfirmCount = deleteConfirmItems.length;

  const openBatchDeleteConfirm = useCallback(() => {
    if (selectedDeleteCount === 0 || deleteSubmitting) return;

    setDeleteConfirmItems(
      items.filter((item) => batchDeleteIds.includes(item.id)),
    );
  }, [batchDeleteIds, deleteSubmitting, items, selectedDeleteCount]);

  const gotoPrev = useCallback(() => {
    setPage((prev) => Math.max(1, prev - 1));
  }, []);

  const gotoNext = useCallback(() => {
    setPage((prev) => prev + 1);
  }, []);

  const copyResourceLink = async (item: GalleryItem) => {
    if (copyingId !== null) return;
    setCopyingId(item.id);
    try {
      const shares = await getResourceShares(item.id);
      const share = latestPublicShare(shares);
      if (share) {
        await handleCopy(shareUrl(share.code));
      } else {
        setShare(item);
        addToast({ title: "暂无可用的无密码分享，请先创建", color: "warning" });
      }
    } catch (error) {
      if (error instanceof Error) addToast({ title: error.message, color: "danger" });
    } finally {
      setCopyingId(null);
    }
  };

  const deletingIdSet = useMemo(() => {
    if (!deleteSubmitting) return new Set<number>();

    return new Set(deleteConfirmItems.map((item) => item.id));
  }, [deleteConfirmItems, deleteSubmitting]);

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
        {items.map((item) => {
          const isBatchSelected = batchDeleteIds.includes(item.id);

          return (
            <div
              key={item.id}
              className={clsx(
                "cursor-pointer rounded-2xl border-2 border-transparent focus:outline-none",
                bathDeleteMode && "transition",
                isBatchSelected &&
                  "border-primary shadow-[0_0_0_3px_rgba(0,111,238,0.18)]",
              )}
              role="button"
              tabIndex={0}
              onKeyDown={(event) => {
                if (event.target !== event.currentTarget || (event.key !== "Enter" && event.key !== " ")) return;
                event.preventDefault();
                if (bathDeleteMode) toggleBatchDeleteItem(item);
                else setPreview(item);
              }}
              onClick={() => {
                if (bathDeleteMode) {
                  toggleBatchDeleteItem(item);

                  return;
                }

                setPreview(item);
              }}
            >
              <GalleryCard
                deleting={deletingIdSet.has(item.id)}
                item={item}
                copying={copyingId === item.id}
                onCopyLink={() => void copyResourceLink(item)}
                onDelete={(target) => setDeleteConfirmItems([target])}
                onShare={() => setShare(item)}
              />
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3 pt-2">
        <div>
          <CheckboxGroup
            className="gap-1"
            label=""
            orientation="horizontal"
            value={tagSelected}
            onChange={(values: any) => handleTagChange([...values])}
          >
            {availableTags.map((tag) => (
              <BillCheckbox key={tag} value={tag}>
                {tag}
              </BillCheckbox>
            ))}
          </CheckboxGroup>
        </div>
        <div className="flex items-center gap-3 text-sm text-default-500">
          <Button
            isLoading={loading}
            color="danger"
            isDisabled={deleteSubmitting}
            size="sm"
            variant="flat"
            onPress={() => {
              setBathDeleteMode((prev) => !prev);
            }}
          >
            {bathDeleteMode ? "退出批量删除" : "批量删除"}
          </Button>
          {bathDeleteMode ? (
            <Button
              color="danger"
              isDisabled={selectedDeleteCount === 0 || deleteSubmitting}
              size="sm"
              variant="flat"
              onPress={openBatchDeleteConfirm}
            >
              删除 {selectedDeleteCount} 个
            </Button>

          ) : (<>
            <span>共 {total} 个</span>
            <span>
              第 {Math.min(page, totalPages)} / {totalPages} 页
            </span>
            <div className="flex items-center gap-2">
              <span className="whitespace-nowrap">每页</span>
              <Select
                aria-label="每页数量"
                className="w-22"
                isDisabled={loading}
                selectedKeys={new Set([String(pageSize)])}
                size="sm"
                onSelectionChange={(keys: any) =>
                  handlePageSizeChange(keys.currentKey)
                }
              >
                {PAGE_SIZE_OPTIONS.map((size) => (
                  <SelectItem key={String(size)} textValue={`${size} 条`}>
                    {size} 条
                  </SelectItem>
                ))}
              </Select>
            </div>
            <Button
              isLoading={loading}
              color="primary"
              size="sm"
              variant="flat"
              onPress={() => {
                fetchTags();
                fetchData(page, tagSelected);
              }}
            >
              刷新
            </Button>
          </>)}
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

      {/* delete confirm modal */}
      <Modal
        isDismissable={!deleteSubmitting}
        isOpen={deleteConfirmCount > 0}
        placement="center"
        size="md"
        onOpenChange={(open: boolean) => {
          if (!open && !deleteSubmitting) {
            setDeleteConfirmItems([]);
          }
        }}
      >
        <ModalContent>
          {(close: () => void) =>
            deleteConfirmCount > 0 ? (
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
                    {deleteConfirmCount === 1
                      ? `将删除资源：${deleteConfirmItems[0]?.filename}`
                      : `将删除选中的 ${deleteConfirmCount} 个资源。`}
                  </p>
                </ModalBody>
                <ModalFooter>
                  <Button
                    color="default"
                    isDisabled={deleteSubmitting}
                    variant="flat"
                    onPress={close}
                  >
                    取消
                  </Button>
                  <Button
                    color="danger"
                    isLoading={deleteSubmitting}
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

      {preview && (
        <ResourcePreviewModal
          key={preview.id}
          resource={preview}
          onClose={() => setPreview(null)}
          onReplaced={(updated) => {
            setPreview((current) => current?.id === updated.id ? { ...updated, tags: current.tags } : current);
            setItems((current) => current.map((item) => item.id === updated.id ? { ...updated, tags: item.tags } : item));
          }}
          onTagsChanged={(tags) => {
            setPreview((current) => current?.id === preview.id ? { ...current, tags } : current);
            setItems((current) => current.map((item) => item.id === preview.id ? { ...item, tags } : item));
            void fetchTags();
            void fetchData(page);
          }}
        />
      )}
      {share && (
        <ShareFormModal resource={share} onClose={() => setShare(null)} onSaved={() => {}} />
      )}
    </div>
  );
}
