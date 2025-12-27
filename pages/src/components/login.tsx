import { useEffect, useState } from "react";
import { Button, Input, Modal, ModalBody, ModalContent, ModalFooter, ModalHeader, useDisclosure, addToast } from "@heroui/react";

import api, { ApiResponse } from "@/lib/api";
import { UserProfile } from "@/types/user";

export default function Login() {
  const [user, setUser] = useState<UserProfile | null>(null);
  const { isOpen, onOpen, onOpenChange, onClose } = useDisclosure();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const btnClassName = "text-default-600 justify-start data-[hover=true]:text-black";

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const data = await api.get<UserProfile>("/me");
        setUser(data);
      } catch {
        setUser(null);
      }
    };
    fetchUser();
  }, []);

  const loginHandler = async () => {
    if (!username.trim() || !password.trim()) return;
    try {
      const profile = await api.post<UserProfile>("/login", {
        username: username.trim(),
        password: password.trim(),
      });
      setUser(profile);
      onClose();
      addToast({ title: "登录成功", color: "success", variant: "flat" });
    } catch (err) {
      addToast({
        title: "登录失败",
        description: (err as ApiResponse<unknown>).msg,
        color: "danger",
        variant: "flat",
      });
    }
  };

  const logoutHandler = async () => {
    try {
      await api.post("/logout");
      setUser(null);
      window.location.reload();
    } catch (err) {
      addToast({
        title: "退出失败",
        description: getApiErrorMessage(err),
        color: "danger",
        variant: "flat",
      });
    }
  };

  return (
    <>
      {user ? (
        <Button className={btnClassName} startContent={null} variant="light" onPress={logoutHandler}>
          {user.nickname || user.username} 退出登录
        </Button>
      ) : (
        <Button color="primary" size="sm" startContent={null} variant="ghost" onPress={onOpen}>
          登录
        </Button>
      )}

      <Modal isDismissable={false} isKeyboardDismissDisabled isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {() => (
            <>
              <ModalHeader className="flex flex-col gap-1">登录</ModalHeader>
              <ModalBody>
                <div className="space-y-3">
                  <Input label="Email" name="username" placeholder="请输入邮箱或用户名" type="text" onValueChange={(value) => setUsername(value)} />
                  <Input label="Password" name="password" placeholder="请输入密码" type="password" onValueChange={(value) => setPassword(value)} />
                </div>
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>
                  取消
                </Button>
                <Button color="primary" onPress={loginHandler}>
                  登录
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </>
  );
}
