package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"linkit/internal/config"
	"linkit/internal/db"
)

// handleCLI 用于处理仅在 CLI 模式下运行的命令。
func handleCLI(cfg config.Config, logger *slog.Logger, args []string) (bool, error) {
	if len(args) == 0 {
		return true, nil
	}
	key := args[0]
	switch key {
	case "server":
		return true, parseServerRunArgs(args)
	case "reset-password":
		if err := resetAdminPassword(cfg, logger, args[1]); err != nil {
			return false, err
		}
		return false, nil
	default:
		return false, fmt.Errorf("未知命令")
	}
}

func parseServerRunArgs(args []string) error {
	if len(args) == 1 && args[0] == "server" {
		return nil
	}
	if args[1] != "run" {
		return fmt.Errorf("使用 linkit server run 启动服务")
	}
	return nil
}

func resetAdminPassword(cfg config.Config, logger *slog.Logger, newPassword string) error {
	password := strings.TrimSpace(newPassword)
	if password == "" {
		return fmt.Errorf("新密码不能为空")
	}
	store, err := db.NewStore(cfg, logger, false)
	if err != nil {
		return err
	}
	defer store.Close()

	ctx, cancel := store.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	adminUser, err := store.User.FindByCredential(ctx, cfg.AdminEmail)
	if err != nil {
		return err
	}
	if adminUser == nil {
		adminUser, err = store.User.FindByCredential(ctx, cfg.AdminUsername)
		if err != nil {
			return err
		}
	}
	if adminUser == nil {
		return fmt.Errorf("请先启动项目，默认密码为 123123")
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := store.User.UpdatePassword(ctx, adminUser.ID, string(pwHash)); err != nil {
		return err
	}
	if err := store.User.UpdateToken(ctx, adminUser.ID, nil); err != nil {
		return err
	}

	logger.Info("管理员密码重置成功", "user", adminUser.Username)
	return nil
}
