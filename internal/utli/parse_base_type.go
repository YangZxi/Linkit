package utli

import (
	"log"
	"strconv"
	"strings"
)

func FirstValue(values []string, fallback string) string {
	if len(values) > 0 && values[0] != "" {
		return values[0]
	}
	return fallback
}

func ParseInt64(value string, fallback int64) int64 {
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return n
	}
	return fallback
}

func ParseOptionalInt64(value string) *int64 {
	if value == "" {
		return nil
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return &n
	}
	return nil
}

func ParseOptionalBool(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	bol, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("解析布尔值失败: %s", value)
		return false
	}
	return bol
}
