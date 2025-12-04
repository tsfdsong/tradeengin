package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tsfdsong/tradeengin/app/gateway/internal/config"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/handler"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	logx.MustSetup(logx.LogConf{Stat: false, Encoding: "plain"})

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 创建服务器
	server := rest.MustNewServer(c.RestConf)

	// 注册优雅关闭 - 使用go-zero的proc包
	proc.AddShutdownListener(func() {
		logx.Info("Shutting down gateway server...")
		server.Stop()
	})

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// 注册健康检查端点
	if c.HealthCheck.Enabled {
		registerHealthCheck(server, c, ctx)
	}

	// 监听系统信号
	go handleSignals(server)

	fmt.Printf("Starting gateway server at %s:%d...\n", c.Host, c.Port)
	logx.Infof("Gateway server starting with config: RateLimit=%v, Breaker=%v, HealthCheck=%v",
		c.RateLimit.Enabled, c.Breaker.Enabled, c.HealthCheck.Enabled)

	server.Start()
}

// registerHealthCheck 注册健康检查端点
func registerHealthCheck(server *rest.Server, c config.Config, ctx *svc.ServiceContext) {
	// 就绪检查 - 检查服务是否可以接收请求
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    c.HealthCheck.Path,
		Handler: healthHandler(ctx),
	})

	// 存活检查 - 检查服务进程是否存活
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    c.HealthCheck.LivePath,
		Handler: liveHandler(),
	})

	logx.Infof("Health check endpoints registered: %s, %s", c.HealthCheck.Path, c.HealthCheck.LivePath)
}

// healthHandler 就绪检查处理器
func healthHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"checks":    make(map[string]string),
		}

		checks := response["checks"].(map[string]string)

		// 检查Redis连接
		if ctx.RedisClient != nil {
			if ok := ctx.RedisClient.Ping(); !ok {
				checks["redis"] = "unhealthy"
				response["status"] = "degraded"
			} else {
				checks["redis"] = "healthy"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if response["status"] == "ok" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		// 简单的JSON输出
		fmt.Fprintf(w, `{"status":"%s","timestamp":%d}`, response["status"], response["timestamp"])
	}
}

// liveHandler 存活检查处理器
func liveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"alive","timestamp":%d}`, time.Now().Unix())
	}
}

// handleSignals 处理系统信号
func handleSignals(server *rest.Server) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs
	logx.Infof("Received signal: %v", sig)

	// 优雅关闭
	server.Stop()
}
