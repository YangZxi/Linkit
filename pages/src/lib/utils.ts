/**
 * 复制文本，兼容非 https 环境。
 */
export async function copyText(text: string): Promise<boolean> {
  if (!text) return false;

  const fallbackCopy = () => {
    try {
      const textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.select();
      const ok = document.execCommand("copy");
      document.body.removeChild(textarea);
      return ok;
    } catch {
      return false;
    }
  };

  try {
    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }

    return fallbackCopy();
  } catch {
    return fallbackCopy();
  }
}
