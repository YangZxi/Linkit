import path from "path-browserify";

export type MediaType = "image" | "video" | "audio" | "text" | "pdf" | "other";

export const TypeLabel: Record<MediaType, string> = {
  image: "图片",
  video: "视频",
  audio: "音频",
  text: "文本类文件",
  pdf: "PDF",
  other: "文件",
};

export function inferMediaType(mime: string): MediaType {
  if (!mime) return "other";
  const lower = mime.toLowerCase();
  if (lower.startsWith("image/")) return "image";
  if (lower.startsWith("video/")) return "video";
  if (lower.startsWith("audio/")) return "audio";
  if (lower.startsWith("text/")) return "text";
  if (lower.includes("pdf")) return "pdf";
  return "other";
}

export const FileTypeTable = {
  image: ["png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "tiff", "ico"],
  video: ["mp4", "mov", "avi", "mkv", "webm", "flv", "wmv"],
  audio: ["mp3", "wav", "aac", "flac", "ogg", "m4a"],
  text: [
    "text",
    "txt",
    "md",
    "json",
    "xml",
    "csv",
    "log",
    "html",
    "css",
    "js",
    "ts",
    "jsx",
    "tsx",
    "c",
    "cpp",
    "java",
    "py",
    "rb",
    "php",
    "sh",
    "yaml",
    "yml",
    "ini",
    "cfg",
  ],
};
