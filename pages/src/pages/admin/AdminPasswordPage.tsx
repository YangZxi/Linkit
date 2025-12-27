import { useState } from "react";
import { Button, Card, CardBody, CardHeader, Input, addToast } from "@heroui/react";

import api, { ApiResponse } from "@/lib/api";

export default function AdminPasswordPage() {
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword1, setNewPassword1] = useState("");
  const [newPassword2, setNewPassword2] = useState("");
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    if (!oldPassword.trim() || !newPassword1.trim() || !newPassword2.trim()) {
      addToast({ title: "请填写完整", color: "warning", variant: "flat" });
      return;
    }
    if (newPassword1 !== newPassword2) {
      addToast({ title: "两次新密码不一致", color: "warning", variant: "flat" });
      return;
    }

    setSaving(true);
    try {
      await api.post("/admin/password", {
        oldPassword,
        newPassword: newPassword1,
        newPassword2,
      });
      setOldPassword("");
      setNewPassword1("");
      setNewPassword2("");
      addToast({ title: "密码修改成功", color: "success", variant: "flat" });
    } catch (err) {
      addToast({
        title: "密码修改失败",
        description: (err as ApiResponse<unknown>).msg,
        color: "danger",
        variant: "flat",
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <Card className="border border-default-200/70 bg-background/70 shadow-sm">
        <CardHeader className="flex flex-col items-start gap-1">
          <p className="text-base font-semibold">修改密码</p>
          <p className="text-sm text-default-500">输入原密码与新密码。</p>
        </CardHeader>
        <CardBody className="space-y-4">
          <Input
            label="原密码"
            placeholder="请输入原密码"
            type="password"
            value={oldPassword}
            onValueChange={setOldPassword}
          />
          <Input
            label="新密码"
            placeholder="请输入新密码"
            type="password"
            value={newPassword1}
            onValueChange={setNewPassword1}
          />
          <Input
            label="确认新密码"
            placeholder="请再次输入新密码"
            type="password"
            value={newPassword2}
            onValueChange={setNewPassword2}
          />
          <div className="flex justify-end">
            <Button color="primary" isLoading={saving} onPress={submit}>
              保存
            </Button>
          </div>
        </CardBody>
      </Card>
    </div>
  );
}
