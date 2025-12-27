import { useCallback, useEffect, useRef, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { Button, Input, Spinner } from "@heroui/react";

import SharePreview from "@/components/share-preview";
import { inferMediaType } from "@/lib/file";
import api, { ApiResponse } from "@/lib/api";

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

type ShareState = {
  status: "loading" | "first" | "success" | "error";
  data?: ShareInfoResponse;
  msg?: string;
  code?: number;
  pwd?: string;
};

export default function SharePage() {
  const { code } = useParams<{ code: string }>();
  const [searchParams, _] = useSearchParams();
  const urlPwd = searchParams.get("pwd") || "";
  const [password, setPassword] = useState(urlPwd);
  const [shareState, setShareState] = useState<ShareState>({
    status: "loading",
  });
  const inputRef = useRef<HTMLInputElement>(null);

  const loadShareInfo = () => {
    setShareState({ status: "loading" });
    const query = password ? `?pwd=${encodeURIComponent(password)}` : "";
    const res = api.get<ShareInfoResponse>(`/share/${code}${query}`);
    setTimeout(() => {
      res.then((data) => {
        history.replaceState({}, "", `/s/${code}${query}`);
        setShareState({ status: "success", data, pwd: password });
      }).catch((err: ApiResponse<unknown>) => {
        // 首次进入
        if (!password && (err.code === 401 || err.code === 403)) {
          setShareState({ status: "first" });
          return;
        }
        if (urlPwd) {
          history.replaceState({}, "", `/s/${code}`);
        }
        setShareState({ status: "error", msg: err.msg, code: err.code });
        setTimeout(() => {
          inputRef.current?.select();
        }, 100);
      })
    }, 300);
  }

  useEffect(() => {
    if (!code) {
      setShareState({ status: "error", msg: "资源不存在或已失效" });
      return;
    }
    loadShareInfo();
  }, [code, urlPwd]);

  const submitPassword = () => {
    if (!password) {
      setShareState({ status: "error", msg: "请输入分享密码" });
      return;
    }
    loadShareInfo();
  };

  if (shareState.status !== "success") {
    return (
      <div className="flex min-h-[300px] flex-col items-center justify-center gap-4 text-default-500">
        <p>该分享需要密码访问</p>
        <div className="w-full max-w-[300px]">
          <div className="h-[64px] w-full">
            <Input
              ref={inputRef}
              autoFocus
              isClearable
              isDisabled={shareState.status === "loading"}
              isInvalid={shareState.status === "error"}
              errorMessage={shareState.msg ?? undefined}
              label="密码"
              labelPlacement="outside"
              type="text"
              value={password}
              onValueChange={(val) => setPassword(val.trim())}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  submitPassword();
                }
              }}
            />
          </div>
          <div className="w-full text-right">
            <Button isLoading={shareState.status === "loading"} color="primary" variant="flat" onPress={submitPassword}>
              确认
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (!shareState.data && shareState.code === 404) {
    return (
      <div className="flex min-h-[300px] flex-col items-center justify-center gap-3 text-default-500">
        <p>资源不存在或已失效</p>
        {shareState.msg && (
          <p className="text-xs text-danger-500">{shareState.msg}</p>
        )}
      </div>
    );
  }

  const type = inferMediaType(shareState.data?.type || "");
  const rawUrl = shareState.pwd
    ? `/r/${code}?pwd=${encodeURIComponent(shareState.pwd)}`
    : `/r/${code}`;
  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-8 my-2 md:my-6 md:px-4">
      <SharePreview
        code={code ?? ""}
        filename={shareState.data?.filename ?? ""}
        rawUrl={rawUrl}
        type={type}
      />
    </div>
  );
}
