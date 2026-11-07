export type UploadFormRequest = {
  file: File;
  filename: string;
  filesize: number;
  uploadId: string;
  chunkIndex: number | null;
  totalChunks: number | null;
  chunkSize: number | null;
};

export type UploadChunkResponse = {
  merged: boolean;
  uploadId: string;
  filename: string;
  size?: number;
  skipped?: boolean;
  chunkIndex?: number | null;
  totalChunks?: number | null;
  chunkSize?: number | null;
};

export type UploadCompletedResponse = {
  resourceId: number;
  shareCode: string;
  filename: string;
  size: number;
  uploadId: string;
};

export type GalleryItem = {
  id: number;
  filename: string;
  type: string;
  storage: "local" | "s3";
  createdAt: string;
  contentVersion: string;
  tags: string[];
};

export type GalleryResponse = {
  data: GalleryItem[];
  total: number;
  page: number;
};

export type GalleryTagsResponse = {
  tags: string[];
};

export type GalleryDeleteResponse = {
  success: boolean;
};

export type CreateShareResponse = {
  code: string;
  expireTime: string | null;
};

export type ManagedShare = {
  id: number;
  code: string;
  password: string | null;
  expireTime: string | null;
  relay: boolean;
  viewCount: number;
  createdAt: string;
};
