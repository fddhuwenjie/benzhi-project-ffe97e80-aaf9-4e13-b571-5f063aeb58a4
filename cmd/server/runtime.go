package main

import (
	"context"
	"dialect-release/internal/application"
	"dialect-release/internal/httpapi"
	"dialect-release/internal/store"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type runtime struct {
	repo     *store.SQLiteRepository
	server   *http.Server
	listener net.Listener
}

func start(cfg config) (*runtime, error) {
	repo, err := store.Open(cfg.Database)
	if err != nil {
		return nil, err
	}
	api := httpapi.New(application.New(repo))
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		repo.Close()
		return nil, fmt.Errorf("监听 %s: %w", cfg.Address, err)
	}
	server := &http.Server{Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	r := &runtime{repo: repo, server: server, listener: listener}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("HTTP 服务异常退出: %v\n", err)
		}
	}()
	return r, nil
}

func (r *runtime) close(ctx context.Context) error {
	serverErr := r.server.Shutdown(ctx)
	dbErr := r.repo.Close()
	if serverErr != nil {
		return serverErr
	}
	return dbErr
}
