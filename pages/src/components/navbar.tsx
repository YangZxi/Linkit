import { Link as RouterLink, useLocation } from "react-router-dom";
import { Navbar, NavbarBrand, NavbarContent, NavbarItem, Link, link as linkStyles } from "@heroui/react";
import clsx from "clsx";

import Login from "./login";
import { GithubIcon, Logo } from "@/components/icons";
import { ThemeSwitch } from "@/components/theme-switch";
import { siteConfig } from "@/config/site";

function AppNavbar() {
  const location = useLocation();

  return (
    <Navbar
      className="bg-transparent"
      classNames={{
        base: [
          "md:min-w-120 top-[10px] md:top-[20px]",
          "z-50 w-auto mx-auto inset-x-0",
          "bg-background/70 backdrop-blur-xl backdrop-saturate-150",
          "rounded-full border border-default-200/50 shadow-lg",
          "max-w-fit px-4",
          "data-[menu-open=true]:border-none data-[menu-open=true]:bg-transparent",
        ],
        wrapper: "px-0 gap-6 h-12",
        item: "data-[active=true]:text-primary data-[active=true]:font-medium",
      }}
      maxWidth="full"
    >
      <NavbarContent className="basis-1/5 sm:basis-full" justify="start">
        <NavbarBrand as="li" className="gap-3 max-w-fit mr-2">
          <RouterLink className="flex justify-start items-center gap-1" to="/">
            <Logo />
            <p className="font-bold text-inherit hidden sm:block">{siteConfig.name}</p>
          </RouterLink>
        </NavbarBrand>
        <ul className="flex gap-6 justify-start ml-2">
          {siteConfig.navItems.map((item) => {
            const active = location.pathname === item.href;
            return (
              <NavbarItem key={item.href} isActive={active}>
                <Link
                  as={RouterLink}
                  className={clsx(
                    linkStyles({ color: "foreground" }),
                    `${item.mobile === true ? "" : "hidden sm:inline-block"}`,
                    "data-[active=true]:text-primary data-[active=true]:font-medium opacity-80 hover:opacity-100 transition-opacity",
                  )}
                  color="foreground"
                  to={item.href}
                >
                  {item.label}
                </Link>
              </NavbarItem>
            );
          })}
        </ul>
      </NavbarContent>

      <NavbarContent className="sm:flex basis-1/5 sm:basis-full" justify="end">
        <div className="h-6 w-[1px] bg-default-300 mx-1 hidden sm:block" />
        <NavbarItem className="flex gap-2">
          <Link
            isExternal
            aria-label="Github"
            className="hidden md:block text-default-500 hover:text-foreground transition-colors"
            href="https://github.com/yangzxi/linkit"
          >
            <GithubIcon />
          </Link>
          <ThemeSwitch />
        </NavbarItem>

        <NavbarItem>
          <Login />
        </NavbarItem>
      </NavbarContent>
    </Navbar>
  );
}

export default AppNavbar;
