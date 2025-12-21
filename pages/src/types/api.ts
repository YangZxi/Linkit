export type UploadFormRequest = {
  file: File;
  fileName: string;
  fileSize: number;
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
  createdAt: string;
  shareCode: string | null;
};

export type GalleryResponse = {
  data: GalleryItem[];
  total: number;
  page: number;
};

export type GalleryDeleteResponse = {
  success: boolean;
};
