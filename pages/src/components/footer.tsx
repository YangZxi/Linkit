import { Link } from "@heroui/react";

function Footer() {
  return (
    <footer
      className="
        flex flex-wrap h-[var(--footer-height)] px-4 py-3
        items-center justify-center border-t border-default-200/60
        bg-background/70 bg-transparent
        backdrop-saturate-150 backdrop-blur dark:border-default-100/20
        rounded-3xl shadow-[0_-10px_30px_rgba(0,0,0,0.06)]
      "
    >
      <a
        className="w-full text-center"
        href="https://github.com/yangzxi/linkit"
        rel="noreferrer"
        target="_blank"
        title="heroui.com homepage"
      >
        <span className="text-default-600">Powered by</span>
        <span className="text-primary font-bold ml-1">
          Linkit
        </span>
      </a>
      <p className="hidden md:block text-sm">
        <span className="hidden md:inline ml-1">
          如果你喜欢本项目，欢迎自建部署或给个⭐
        </span>
      </p>
    </footer>
  );
}

export default Footer;
