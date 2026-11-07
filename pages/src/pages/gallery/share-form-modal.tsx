import { useState } from "react";
import {
  Alert, Button, Input, Modal, ModalBody, ModalContent, ModalFooter, ModalHeader,
  NumberInput, Switch, addToast,
} from "@heroui/react";

import type { CreateShareResponse, GalleryItem, ManagedShare } from "@/types/api";
import api from "@/lib/api";
import { copyText } from "@/lib/utils";
import { formatShareTime, shareCopyText, shareUrl } from "./gallery-api";

type DurationUnit = "minutes" | "hours" | "days";
const UNIT_SECONDS: Record<DurationUnit, number> = { minutes: 60, hours: 3600, days: 86400 };
const MAX_DURATION_SECONDS = 100 * 365 * 86400;

function randomPassword() {
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  return Array.from(crypto.getRandomValues(new Uint8Array(6)), (value) => chars[value % chars.length]).join("");
}

type Props = {
  resource: GalleryItem;
  share?: ManagedShare | null;
  onClose: () => void;
  onSaved: () => void;
};

export default function ShareFormModal({ resource, share, onClose, onSaved }: Props) {
  const editing = Boolean(share);
  const [code, setCode] = useState(share?.code || "");
  const [passwordEnabled, setPasswordEnabled] = useState(share ? Boolean(share.password) : true);
  const [password, setPassword] = useState(() => share?.password || randomPassword());
  const [duration, setDuration] = useState(0);
  const [unit, setUnit] = useState<DurationUnit>("days");
  const [changeExpiration, setChangeExpiration] = useState(!editing);
  const [relay, setRelay] = useState(share?.relay || false);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<(CreateShareResponse & { password: string | null }) | null>(null);
  const [error, setError] = useState("");

  const seconds = Math.round(duration * UNIT_SECONDS[unit]);
  const durationValid = Number.isFinite(seconds) && seconds >= 0 && seconds <= MAX_DURATION_SECONDS;
  const expiration = result ? result.expireTime : durationValid && seconds > 0
    ? new Date(Date.now() + seconds * 1000).toISOString()
    : null;
  const expirationLabel = expiration
    ? `有效期（${formatShareTime(expiration)} 到期）`
    : "有效期 (为 0 表示永不过期)";
  const disabled = submitting || Boolean(result);

  const submit = async () => {
    if (submitting) return;
    if (result) {
      const copied = await copyText(shareCopyText(result));
      addToast({ title: copied ? "已复制分享信息" : "复制失败，请手动复制", color: copied ? "success" : "danger" });
      return;
    }
    const trimmedCode = code.trim();
    const trimmedPassword = passwordEnabled ? password.trim() : "";
    if (!editing && trimmedCode && !/^[a-zA-Z0-9]{4,32}$/.test(trimmedCode)) {
      setError("分享短码需为 4–32 位英文字母或数字");
      return;
    }
    if (passwordEnabled && ([...trimmedPassword].length < 4 || [...trimmedPassword].length > 32)) {
      setError("分享密码长度需为 4–32 位");
      return;
    }
    if (changeExpiration && !durationValid) {
      setError("请输入有效的有效期，最长为 100 年");
      return;
    }
    setError("");
    setSubmitting(true);
    try {
      const settings = {
        password: trimmedPassword,
        relay,
        ...(changeExpiration ? { durationSeconds: seconds } : {}),
      };
      if (share) {
        await api.post(`/gallery/${resource.id}/shares/${share.id}/update`, settings, { hideToast: true });
        addToast({ title: "分享设置已保存", color: "success" });
        onSaved();
        onClose();
      } else {
        const created = await api.post<CreateShareResponse>("/share", {
          resourceId: resource.id, code: trimmedCode, ...settings,
        }, { hideToast: true });
        setResult({ ...created, password: trimmedPassword || null });
        onSaved();
      }
    } catch (cause) {
      setError((cause as { msg?: string; message?: string } | null)?.msg ||
        (cause as Error | null)?.message || "保存失败，请稍后重试");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal isOpen isDismissable={!submitting} isKeyboardDismissDisabled={submitting} hideCloseButton={submitting}
      placement="center" scrollBehavior="inside" onClose={onClose}>
      <ModalContent>
        <form className="contents" onSubmit={(event) => { event.preventDefault(); void submit(); }}>
          <ModalHeader className="flex flex-col gap-1 pr-10">
            <span>{editing ? "编辑分享" : "创建分享"}</span>
            <span className="truncate text-sm font-normal text-default-500">{resource.filename}</span>
          </ModalHeader>
          <ModalBody className="gap-4">
            <Input autoFocus={!editing} label="分享短码" placeholder="留空则随机生成"
              description={editing ? "短码保持不变，需要更换请创建新分享" : "4–32 位英文字母或数字，区分大小写"}
              isReadOnly={editing} isDisabled={disabled} maxLength={32}
              value={code} onValueChange={setCode} />
            <div className="flex items-center gap-3">
              <Switch aria-label="启用分享密码" size="sm" isSelected={passwordEnabled}
                isDisabled={disabled} onValueChange={setPasswordEnabled} />
              <Input className="min-w-0 flex-1" label="分享密码 (4–32位)" type="text"
                value={passwordEnabled ? password : ""} placeholder={passwordEnabled ? "请输入密码" : "无需密码"}
                isDisabled={disabled || !passwordEnabled} onValueChange={setPassword} />
            </div>
            {editing && (
              <div className="space-y-2">
                <p className="text-sm text-default-500">
                  当前有效期：{share?.expireTime ? `${formatShareTime(share.expireTime)} 到期` : "永不过期"}
                </p>
                <Switch size="sm" isSelected={changeExpiration} isDisabled={disabled}
                  onValueChange={setChangeExpiration}>重新设置有效期</Switch>
              </div>
            )}
            {changeExpiration && (
              <NumberInput label={expirationLabel} value={duration} minValue={0}
                maxValue={MAX_DURATION_SECONDS / UNIT_SECONDS[unit]} isDisabled={disabled}
                onValueChange={setDuration} description={editing ? "从本次保存时重新计时" : undefined}
                endContent={
                  <select aria-label="有效期单位" className="border-0 bg-transparent text-sm text-default-500 outline-none"
                    value={unit} disabled={disabled} onChange={(event) => setUnit(event.target.value as DurationUnit)}>
                    <option value="minutes">分钟</option><option value="hours">小时</option><option value="days">天</option>
                  </select>
                } />
            )}
            <Switch size="sm" isSelected={relay} isDisabled={disabled} onValueChange={setRelay}>
              资源加速（S3 使用服务器转发）
            </Switch>
            {error && <Alert color="danger" title={error} />}
            {result && (
              <Alert color="success" title="分享创建成功">
                <div className="space-y-1 break-all text-sm">
                  <button type="button" className="text-left text-primary underline"
                    onClick={async () => {
                      const copied = await copyText(shareUrl(result.code));
                      addToast({ title: copied ? "已复制分享链接" : "复制失败，请手动复制", color: copied ? "success" : "danger" });
                    }}>{shareUrl(result.code)}</button>
                  <p>{result.password ? `密码：${result.password}` : "无需密码"}</p>
                </div>
              </Alert>
            )}
          </ModalBody>
          <ModalFooter>
            <Button variant="flat" isDisabled={submitting} onPress={onClose}>关闭</Button>
            <Button color="primary" type="submit" isLoading={submitting}>
              {result ? "复制分享信息" : editing ? "保存" : "创建分享"}
            </Button>
          </ModalFooter>
        </form>
      </ModalContent>
    </Modal>
  );
}
