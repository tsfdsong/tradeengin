
genaccount:
#gormt 通过数据库生成指定的结构体 https://github.com/xxjwxc/gormt -z config.yaml 指定配置文件路径
	gentool --dsn="root:root@tcp(192.168.2.159:3307)/trade?charset=utf8mb4&parseTime=True&loc=Local" --db=mysql --tables=user,asset  -outPath=app/account/rpc/internal/dao/query -fieldMap="decimal:string;tinyint:int32;"
accountapi:
	   goctl api go -api=app/account/api/desc/account.api -dir=app/account/api -style=go_zero  -home=template && make accountdoc
accountdoc:
	   goctl api plugin -plugin goctl-swagger="swagger -filename doc/account.json -host api.gex.com" -api app/account/api/desc/account.api -dir .
accountrpc:
	   goctl rpc  protoc app/account/rpc/pb/account.proto --go_out=app/account/rpc --go-grpc_out=app/account/rpc   --zrpc_out=app/account/rpc -style=go_zero  -home=template
orderrpc:
	   goctl rpc  protoc -Icommon/proto -I./ app/order/rpc/pb/order.proto --go_out=app/order/rpc --go-grpc_out=app/order/rpc   --zrpc_out=app/order/rpc -style=go_zero  -home=template
orderapi:
	   goctl api go -api=app/order/api/desc/order.api -dir=app/order/api -style=go_zero  -home=template && make orderdoc
orderdoc:
	   goctl api plugin -plugin goctl-swagger="swagger -filename doc/order.json -host api.gex.com" -api app/order/api/desc/order.api -dir .
genorder:
	gentool --dsn="root:root@tcp(192.168.2.159:3307)/trade?charset=utf8mb4&parseTime=True&loc=Local" --db=mysql --tables=entrust_order_00,matched_order  -outPath=app/order/rpc/internal/dao/query -fieldMap="decimal:string;tinyint:int32;bigint:int64;"
enum:
	protoc   -I. --go_out=./  common/proto/enum/*.proto
matchmq:
	#--go_out指定的路径和option go_package = "trade/common/proto/mq/match;proto"; 指定的路径一起决定文件生成的位置 这个路径trade/common/proto/mq/match也是别人导入时用到的路径。
	protoc    -Icommon/proto -I./ --go_out=./ common/proto/mq/match/match.proto && make matchmodel
matchrpc:
	goctl rpc  protoc -I./ -Icommon/proto app/match/rpc/pb/match.proto --go_out=app/match/rpc --go-grpc_out=app/match/rpc   --zrpc_out=app/match/rpc -style=go_zero  -home=template
	make matchmodel
matchmodel:
	gentool --dsn="root:root@tcp(192.168.2.159:3307)/trade?charset=utf8mb4&parseTime=True&loc=Local" --db=mysql --tables=matched_order  -outPath=app/match/rpc/internal/dao/query -fieldMap="decimal:string;tinyint:int32;int:int64"
klinerpc:
	goctl rpc  protoc -I./ app/quotes/kline/rpc/pb/kline.proto --go_out=app/quotes/kline/rpc --go-grpc_out=app/quotes/kline/rpc  --zrpc_out=app/quotes/kline/rpc -style=go_zero  -home=template
tickerrpc:
	goctl rpc  protoc -I./ app/quotes/ticker/rpc/pb/ticker.proto --go_out=app/quotes/ticker/rpc --go-grpc_out=app/quotes/ticker/rpc  --zrpc_out=app/quotes/ticker/rpc -style=go_zero  -home=template
depthrpc:
	goctl rpc  protoc -I./ app/quotes/depth/rpc/pb/depth.proto --go_out=app/quotes/depth/rpc --go-grpc_out=app/quotes/depth/rpc  --zrpc_out=app/quotes/depth/rpc -style=go_zero  -home=template

klinemodel:
	gentool --dsn="root:root@tcp(192.168.2.159:3307)/trade?charset=utf8mb4&parseTime=True&loc=Local" --db=mysql --tables=kline  -outPath=app/quotes/kline/rpc/internal/dao/query -fieldMap="decimal:string;tinyint:int32;int:int64"
quoteapi:
	   goctl api go -api=app/quotes/api/desc/quotes.api -dir=app/quotes/api -style=go_zero  -home=template && make quotedoc
quotedoc:
	goctl api plugin -plugin goctl-swagger="swagger -filename doc/quotes.json -host api.gex.com" -api app/quotes/quotes-api/desc/quotes.api -dir .

adminapi:
	goctl api go -api=app/admin/api/desc/admin.api -dir=app/admin/api -style=go_zero  -home=template &&   make admindoc

admindoc:
	goctl api plugin -plugin goctl-swagger="swagger -filename doc/admin.json -host api.gex.com" -api app/admin/api/desc/admin.api -dir .

adminmodel:
	gentool --dsn="root:root@tcp(192.168.2.159:3307)/admin?charset=utf8mb4&parseTime=True&loc=Local" --db=mysql  -outPath=app/admin/api/internal/dao/admin/query -fieldMap="decimal:string;tinyint:int32;int:int32" -fieldSignable=true
	softdeleted -p app/admin/api/internal/dao/model/*.go
	gentool --dsn="root:root@tcp(192.168.2.159:3307)/trade?charset=utf8mb4&parseTime=True&loc=Local" --db=mysql --tables=matched_order  -outPath=app/admin/api/internal/dao/match/query -fieldMap="decimal:string;tinyint:int32;int:int64"

start:
	make pre
	chmod +x ./deploy/scripts/run.sh
	./deploy/scripts/run.sh
stop:
	chmod +x ./deploy/scripts/stop.sh
	./deploy/scripts/stop.sh
clear:
	chmod +x ./deploy/scripts/remove_containers.sh
	chmod +x ./deploy/scripts/remove_images.sh
	./deploy/scripts/remove_containers.sh
	./deploy/scripts/remove_images.sh
	rm -rf deploy/depend/mysql/data/*

pre:
	chmod +x ./bin/gateway
	chmod +x ./bin/matching
	chmod +x ./bin/order

dep1:
	docker-compose -f deploy/depend/docker-compose.yaml up
dep2:
	docker-compose -f deploy/dockerfiles/docker-compose.yaml up

build:
# 	go env -w GOOS=linux
# 	go env -w  GOPROXY=https://goproxy.cn,direct
# 	go env -w  CGO_ENABLED=0
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -ldflags="-s -w"  -o ./bin/gateway ./app/gateway/gateway.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/matching ./app/matching/matching.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/order ./app/order/order.go