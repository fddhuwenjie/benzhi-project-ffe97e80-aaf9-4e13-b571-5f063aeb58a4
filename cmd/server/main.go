package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	if cfg.SelfCheck {
		directory, err := os.MkdirTemp("", "dialect-release-self-check-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(directory)
		cfg.Database = filepath.Join(directory, "self-check.db")
	}
	runtime, err := start(cfg)
	if err != nil {
		return err
	}
	if cfg.SelfCheck {
		checkErr := runSelfCheck(cfg.Address)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		closeErr := runtime.close(ctx)
		if checkErr != nil {
			return fmt.Errorf("self-check 失败: %w", checkErr)
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Println("self-check 通过：授权、材料、脱敏、退回复核、核准和封存链路完整")
		return nil
	}
	fmt.Printf("语守方言语料发布治理服务监听 %s\n", cfg.Address)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return runtime.close(ctx)
}
