import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { Spinner } from "@heroui/react";

import SharePreview from "@/components/share-preview";
import { inferMediaType } from "@/lib/file";
import api from "@/lib/api";

interface ShareInfoResponse {
  share_id: number;
  code: string;
  resource_id: number;
  filename: string;
  path: string;
  type: string;
  viewCount: number;
  created_at: string;
}

export default function SharePage() {
  const { code } = useParams<{ code: string }>();
  const [data, setData] = useState<ShareInfoResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!code) return;
    setLoading(true);
    setError(null);
    api
      .get<ShareInfoResponse>(`/share/${code}`)
      .then((res) => setData(res))
      .catch((err) => setError((err as Error).message || "资源不存在"))
      .finally(() => setLoading(false));
  }, [code]);

  if (loading) {
    return (
      <div className="flex min-h-[300px] items-center justify-center">
        <Spinner color="primary" label="加载中" />
      </div>
    );
  }

  if (error || !data || !code) {
    return (
      <div className="flex min-h-[300px] flex-col items-center justify-center gap-3 text-default-500">
        <p>资源不存在或已失效</p>
        {error && <p className="text-xs text-danger-500">{error}</p>}
      </div>
    );
  }

  const type = inferMediaType(data.type || "");
  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-8 my-2 md:my-6 md:px-4">
      <SharePreview code={code} filename={data.filename} rawUrl={`/r/${code}`} type={type} />
    </div>
  );
}
