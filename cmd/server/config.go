package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type config struct {
	Address   string
	Database  string
	SelfCheck bool
}

func parseConfig(args []string) (config, error) {
	set := flag.NewFlagSet("dialect-release", flag.ContinueOnError)
	address := set.String("addr", "127.0.0.1:19081", "HTTP 监听地址")
	database := set.String("db", "dialect-release.db", "SQLite 数据库路径")
	selfCheck := set.Bool("self-check", false, "运行完整 HTTP 自检后退出")
	if err := set.Parse(args); err != nil {
		return config{}, err
	}
	explicitAddress := false
	set.Visit(func(f *flag.Flag) {
		if f.Name == "addr" {
			explicitAddress = true
		}
	})
	if !explicitAddress {
		if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
			number, err := strconv.Atoi(port)
			if err != nil || number < 1 || number > 65535 {
				return config{}, errors.New("PORT 必须是有效端口号")
			}
			*address = net.JoinHostPort("127.0.0.1", port)
		}
	}
	if err := validateAddress(*address); err != nil {
		return config{}, err
	}
	if strings.TrimSpace(*database) == "" {
		return config{}, errors.New("db 路径不能为空")
	}
	return config{Address: *address, Database: *database, SelfCheck: *selfCheck}, nil
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("addr 格式无效: %w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("addr 端口必须位于 1..65535")
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("addr 必须绑定回环地址")
	}
	return nil
}
