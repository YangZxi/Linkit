import { Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter } from "@heroui/react";
import type { ModalProps } from "@heroui/modal";

type XModalProps = ModalProps & {
  header?: string | React.ReactNode;
  footer?: React.ReactNode;
  children: React.ReactNode;
  submitText?: string;
  onSubmit?: () => void;
}

export default function XModal({
  header, children, footer,
  submitText, onSubmit,
  ...props
}: XModalProps) {


  const renderContent = (close: () => void) => {
    if (onSubmit) {
      footer = <>
        <Button variant="flat" color="default" onPress={close}>关闭</Button>
        <Button variant="flat" color="primary" onPress={onSubmit}>{submitText || "提交"}</Button>
      </>
    }
    return <>
      {header && <ModalHeader className="flex flex-col gap-1 text-default-900 dark:text-default-700">
        {header}
      </ModalHeader>
      }
      <ModalBody>
        {children}
      </ModalBody>
      <ModalFooter>
        {footer ? <>
          {footer}
        </> : <>
          <Button variant="flat" color="primary" onPress={close}>知道了</Button>
        </>}
      </ModalFooter>
    </>
  }

  return (<Modal
    {...props}
    className=""
  >
    <ModalContent>
      {(close) => renderContent(close)}
    </ModalContent>
  </Modal>);
}