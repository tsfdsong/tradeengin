package main

import (
	"flag"
	"fmt"

	"github.com/tsfdsong/tradeengin/app/matching/internal/config"
	"github.com/tsfdsong/tradeengin/app/matching/internal/server"
	"github.com/tsfdsong/tradeengin/app/matching/internal/svc"
	"github.com/tsfdsong/tradeengin/app/matching/match"
	"github.com/tsfdsong/tradeengin/app/pkg/interceptor/rpcserver"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/matching.yaml", "the config file")

func main() {
	flag.Parse()

	logx.MustSetup(logx.LogConf{Stat: false, Encoding: "plain"})

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		match.RegisterMatchServiceServer(grpcServer, server.NewMatchServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	//rpc log
	s.AddUnaryInterceptors(rpcserver.LoggerInterceptor)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
