import type { GalleryItem, ManagedShare } from "@/types/api";

import api, { apiUrl, type ApiResponse } from "@/lib/api";

export function resourceFileUrl(item: GalleryItem) {
  return apiUrl(`/gallery/${item.id}/file?v=${encodeURIComponent(item.contentVersion)}`);
}

export function getResourceShares(resourceId: number) {
  return api.get<ManagedShare[]>(`/gallery/${resourceId}/shares`);
}

export function isShareExpired(share: ManagedShare, now = Date.now()) {
  return Boolean(share.expireTime && new Date(share.expireTime).getTime() <= now);
}

export function sortShares(shares: ManagedShare[], now = Date.now()) {
  return [...shares].sort((a, b) =>
    Number(isShareExpired(a, now)) - Number(isShareExpired(b, now)) ||
    new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime() || b.id - a.id,
  );
}

export function latestPublicShare(shares: ManagedShare[]) {
  return sortShares(shares).find((share) => !share.password && !isShareExpired(share));
}

export function shareUrl(code: string, raw = false) {
  return new URL(`/${raw ? "r" : "s"}/${code}`, window.location.origin).toString();
}

export function shareCopyText(share: Pick<ManagedShare, "code" | "password">) {
  const url = shareUrl(share.code);
  return share.password ? `分享链接：${url}\n密码：${share.password}` : url;
}

export function formatShareTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", { hour12: false });
}

type ReplacementResult = {
  merged: boolean;
  resourceId?: number;
  filename?: string;
  type?: string;
  storage?: GalleryItem["storage"];
  contentVersion?: string;
  warning?: string;
};

// Use the same upload handler and chunk format as ordinary uploads.
export async function replaceResourceFile(
  resource: GalleryItem,
  file: File,
  onProgress: (value: number) => void,
): Promise<{ resource: GalleryItem; warning?: string }> {
  const chunkSize = 8 * 1024 * 1024;
  const totalChunks = Math.max(1, Math.ceil(file.size / chunkSize));
  const uploadId = crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`;

  for (let index = 0; index < totalChunks; index += 1) {
    const start = index * chunkSize;
    const chunk = totalChunks === 1 ? file : file.slice(start, start + chunkSize);
    const form = new FormData();
    form.append("file", chunk, file.name);
    form.append("filename", file.name);
    form.append("filesize", String(file.size));
    form.append("uploadId", uploadId);
    if (totalChunks > 1) {
      form.append("chunkIndex", String(index));
      form.append("totalChunks", String(totalChunks));
      form.append("chunkSize", String(chunkSize));
    }

    const result = await new Promise<ReplacementResult>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open("POST", apiUrl(`/gallery/${resource.id}/file`));
      xhr.withCredentials = true;
      xhr.responseType = "json";
      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable) {
          const fraction = event.loaded / event.total;
          const uploaded = start + fraction * chunk.size;
          onProgress(Math.min(99, Math.round(uploaded / Math.max(1, file.size) * 100)));
        }
      };
      xhr.onload = () => {
        const response = xhr.response as ApiResponse<ReplacementResult> | null;
        if (xhr.status >= 200 && xhr.status < 300 && response?.code === 200) {
          resolve(response.data);
        } else {
          reject(new Error(response?.msg || "文件替换失败"));
        }
      };
      xhr.onerror = () => reject(new Error("网络错误，请检查连接后重试"));
      xhr.onabort = () => reject(new Error("上传已取消"));
      xhr.send(form);
    });

    if (result.merged && result.resourceId === resource.id && result.contentVersion) {
      onProgress(100);
      return {
        resource: {
          ...resource,
          filename: result.filename || file.name,
          type: result.type || "application/octet-stream",
          storage: result.storage || resource.storage,
          contentVersion: result.contentVersion,
        },
        warning: result.warning,
      };
    }
  }
  throw new Error("服务器未确认文件替换完成，请刷新后检查");
}
