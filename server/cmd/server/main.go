package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"chengzi-agent-code-server/internal/config"
	"chengzi-agent-code-server/internal/handler"
	adelogic "chengzi-agent-code-server/internal/logic/ade"
	"chengzi-agent-code-server/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("config", "etc/ade.yaml", "config file path")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	if err := c.Validate(); err != nil {
		panic(err)
	}

	server := rest.MustNewServer(c.RestConf, handler.ServerOptions(c)...)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	defer func() {
		_ = ctx.Shutdown()
	}()

	// ADE 领域服务在 main 装配，避免 svc 与 logic 互相引用。
	adeService := adelogic.NewService(ctx, ctx.AdeRepo, c.Skills.Dir)
	ctx.Ade = adeService
	adelogic.NewScheduler(adeService).Start(context.Background())

	handler.RegisterHandlers(server, ctx)
	handler.RegisterCustomHandlers(server, ctx)
	handler.RegisterAdeHandlers(server, ctx)

	fmt.Printf("ADE server starting at %s:%d (skills: %s)...\n", c.Host, c.Port, c.Skills.Dir)

	go func() {
		server.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down...")
}
