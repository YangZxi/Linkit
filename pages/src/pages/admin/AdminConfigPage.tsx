import { useCallback, useEffect, useState } from "react";
import { Button, Card, CardBody, CardHeader, Input, Select, SelectItem, Spinner, addToast } from "@heroui/react";

import api, { ApiResponse } from "@/lib/api";

type AdminConfigItem = {
  key: string;
  value: string;
  source: "db" | "env" | "default";
  dbValue?: string;
};

type AdminConfigResponse = {
  items: AdminConfigItem[];
};

type ConfigValues = {
  STORAGE_DRIVER?: string;
  S3_BUCKET?: string;
  S3_ACCESS_KEY?: string;
  S3_SECRET_KEY?: string;
  S3_ENDPOINT?: string;
  S3_REGION?: string;
  GUEST_UPLOAD_ENABLE?: string;
  GUEST_UPLOAD_EXT_WHITELIST?: string;
  GUEST_UPLOAD_MAX_MB_SIZE?: string;
};

type ConfigSource = {
  source: AdminConfigItem["source"];
  dbValue?: string;
};

function labelOfSource(source: AdminConfigItem["source"]) {
  switch (source) {
    case "db":
      return "数据库";
    case "env":
      return "环境变量";
    default:
      return "默认值";
  }
}

const defaultValues: ConfigValues = {
  STORAGE_DRIVER: undefined,
  S3_BUCKET: undefined,
  S3_ACCESS_KEY: undefined,
  S3_SECRET_KEY: undefined,
  S3_ENDPOINT: undefined,
  S3_REGION: undefined,
  GUEST_UPLOAD_ENABLE: undefined,
  GUEST_UPLOAD_EXT_WHITELIST: undefined,
  GUEST_UPLOAD_MAX_MB_SIZE: undefined,
};

export default function AdminConfigPage() {
  const [loading, setLoading] = useState(false);
  const [values, setValues] = useState<ConfigValues>(defaultValues);
  const [sources, setSources] = useState<Partial<Record<keyof ConfigValues, ConfigSource>>>({});
  const [saving, setSaving] = useState(false);
  const [submitAttempted, setSubmitAttempted] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get<AdminConfigResponse>("/admin/config");
      const next: ConfigValues = { ...defaultValues };
      const nextSources: Partial<Record<keyof ConfigValues, ConfigSource>> = {};
      for (const it of res.items || []) {
        if (Object.prototype.hasOwnProperty.call(next, it.key)) {
          const key = it.key as keyof ConfigValues;
          next[key] = it.value ?? "";
          nextSources[key] = { source: it.source, dbValue: it.dbValue };
        }
      }
      setValues(next);
      setSources(nextSources);
    } catch (err) {
      setValues(defaultValues);
      setSources({});
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const storageDriver = values.STORAGE_DRIVER?.trim() ?? "";
  const showS3Fields = storageDriver === "s3";
  const storageDriverInvalid = submitAttempted && !storageDriver;
  const guestUploadEnable = values.GUEST_UPLOAD_ENABLE?.trim().toLowerCase() ?? "";
  const guestUploadEnableInvalid =
    submitAttempted && guestUploadEnable !== "true" && guestUploadEnable !== "false";
  const guestUploadMaxMbSize = values.GUEST_UPLOAD_MAX_MB_SIZE?.trim() ?? "";
  const guestUploadMaxMbSizeNumber = Number(guestUploadMaxMbSize);
  const guestUploadMaxMbSizeInvalid =
    submitAttempted &&
    (!guestUploadMaxMbSize ||
      Number.isNaN(guestUploadMaxMbSizeNumber) ||
      !Number.isInteger(guestUploadMaxMbSizeNumber) ||
      guestUploadMaxMbSizeNumber <= 0);
  const sourceHint = (key: keyof ConfigValues) => {
    const meta = sources[key];
    if (!meta) return "当前来源：未知";
    const suffix = meta.dbValue ? `（db=${meta.dbValue}）` : "";
    return `当前来源：${labelOfSource(meta.source)}${suffix}`;
  };

  const save = async () => {
    setSubmitAttempted(true);
    if (!storageDriver) {
      addToast({ title: "StorageDriver 不能为空", color: "danger", variant: "flat" });
      return;
    }
    if (guestUploadEnableInvalid) {
      addToast({ title: "访客上传开关需选择开启或关闭", color: "danger", variant: "flat" });
      return;
    }
    if (guestUploadMaxMbSizeInvalid) {
      addToast({ title: "访客上传大小需为正整数(MB)", color: "danger", variant: "flat" });
      return;
    }
    const newConfigs = { ...values };
    console.log("Saving values:", newConfigs);
    if (newConfigs.STORAGE_DRIVER === "local") {
      // 不更新 S3 相关配置
      newConfigs.S3_BUCKET = undefined;
      newConfigs.S3_ACCESS_KEY = undefined;
      newConfigs.S3_SECRET_KEY = undefined;
      newConfigs.S3_ENDPOINT = undefined;
      newConfigs.S3_REGION = undefined;
    }
  
    setSaving(true);
    try {
      await api.post("/admin/config", {appConfig: newConfigs});
      addToast({ title: "保存成功", color: "success", variant: "flat" });
      await load();
    } catch (err) {
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-default-900">配置管理</h1>
          <p className="mt-1 text-sm text-default-500">仅支持白名单 AppConfig 配置项。</p>
        </div>
        <Button color="primary" isDisabled={loading} isLoading={saving} onPress={save}>
          保存
        </Button>
      </div>

      <Card className="border border-default-200/70 bg-background/70 shadow-sm">
        <CardHeader className="flex items-center justify-between">
          <div>
            <p className="text-base font-semibold">AppConfig</p>
            <p className="mt-1 text-sm text-default-500">优先级：数据库 → 环境变量 → 默认值</p>
          </div>
          <Button isDisabled={loading} size="sm" variant="light" onPress={load}>
            刷新
          </Button>
        </CardHeader>
        <CardBody className="space-y-4">
          {loading ? (
            <div className="flex items-center justify-center py-10">
              <Spinner label="加载中..." />
            </div>
          ) : (
            <div className="flex flex-col gap-4">
              <Select
                errorMessage={storageDriverInvalid ? "StorageDriver 不能为空" : undefined}
                isInvalid={storageDriverInvalid}
                label={`StorageDriver / ${sourceHint("STORAGE_DRIVER")}`}
                placeholder="请选择存储驱动"
                selectedKeys={storageDriver ? new Set([storageDriver]) : new Set([])}
                onSelectionChange={(keys) => {
                  setValues((prev) => ({ ...prev, STORAGE_DRIVER: keys.currentKey ?? "local" }));
                }}
              >
                <SelectItem key="local">本地存储</SelectItem>
                <SelectItem key="s3">S3存储/AWS/Cloudflare</SelectItem>
              </Select>
              {showS3Fields && (
                <>
                  <Input
                    label={`S3Bucket / ${sourceHint("S3_BUCKET")}`}
                    placeholder={"请填写 S3Bucket"}
                    value={values.S3_BUCKET ?? ""}
                    onValueChange={(value) => setValues((prev) => ({ ...prev, S3_BUCKET: value }))}
                  />
                  <Input
                    label={`S3AccessKey / ${sourceHint("S3_ACCESS_KEY")}`}
                    placeholder={"请填写 S3AccessKey"}
                    type="password"
                    value={values.S3_ACCESS_KEY ?? ""}
                    onValueChange={(value) => setValues((prev) => ({ ...prev, S3_ACCESS_KEY: value }))}
                  />
                  <Input
                    label={`S3SecretKey / ${sourceHint("S3_SECRET_KEY")}`}
                    placeholder={"请填写 S3SecretKey"}
                    type="password"
                    value={values.S3_SECRET_KEY ?? ""}
                    onValueChange={(value) => setValues((prev) => ({ ...prev, S3_SECRET_KEY: value }))}
                  />
                  <Input
                    label={`S3Endpoint / ${sourceHint("S3_ENDPOINT")}`}
                    placeholder={"请填写 S3Endpoint"}
                    value={values.S3_ENDPOINT ?? ""}
                    onValueChange={(value) => setValues((prev) => ({ ...prev, S3_ENDPOINT: value }))}
                  />
                  <Input
                    label={`S3Region / ${sourceHint("S3_REGION")}`}
                    placeholder={"请填写 S3Region"}
                    value={values.S3_REGION ?? ""}
                    onValueChange={(value) => setValues((prev) => ({ ...prev, S3_REGION: value }))}
                  />
                </>
              )}
              <div className="pt-2">
                <p className="text-sm font-medium text-default-700">访客(未登录)上传白名单</p>
                <p className="mt-1 text-xs text-default-500">可以设置文件后缀白名单与单文件大小(MB)</p>
              </div>
              <Select
                errorMessage={guestUploadEnableInvalid ? "请选择是否开启访客上传" : undefined}
                isInvalid={guestUploadEnableInvalid}
                label={`访客上传开关 / ${sourceHint("GUEST_UPLOAD_ENABLE")}`}
                placeholder="请选择是否开启"
                selectedKeys={guestUploadEnable ? new Set([guestUploadEnable]) : new Set([])}
                onSelectionChange={(keys) => {
                  setValues((prev) => ({ ...prev, GUEST_UPLOAD_ENABLE: keys.currentKey ?? "false" }));
                }}
              >
                <SelectItem key="false">关闭</SelectItem>
                <SelectItem key="true">开启</SelectItem>
              </Select>
              <Input
                label={`文件后缀白名单，英文逗号分隔 / ${sourceHint("GUEST_UPLOAD_EXT_WHITELIST")}`}
                placeholder={"jpg,jpeg,png,gif"}
                value={values.GUEST_UPLOAD_EXT_WHITELIST ?? ""}
                onValueChange={(value) => setValues((prev) => ({ ...prev, GUEST_UPLOAD_EXT_WHITELIST: value }))}
              />
              <Input
                errorMessage={guestUploadMaxMbSizeInvalid ? "请输入正整数，默认 5" : undefined}
                isInvalid={guestUploadMaxMbSizeInvalid}
                label={`访客上传大小上限(MB) / ${sourceHint("GUEST_UPLOAD_MAX_MB_SIZE")}`}
                placeholder={"5"}
                type="number"
                value={values.GUEST_UPLOAD_MAX_MB_SIZE ?? ""}
                onValueChange={(value) => setValues((prev) => ({ ...prev, GUEST_UPLOAD_MAX_MB_SIZE: value }))}
              />
            </div>            
          )}
        </CardBody>
      </Card>
    </div>
  );
}
