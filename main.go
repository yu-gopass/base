package main

import (
	"github.com/afex/hystrix-go/hystrix"
	ratelimit "github.com/asim/go-micro/plugins/wrapper/ratelimiter/uber/v3"
	"github.com/asim/go-micro/v3"
	"github.com/opentracing/opentracing-go"
	"github.com/yu-gopass/base/handler"
	hystrix2 "github.com/yu-gopass/base/plugin/hystrix"
	base "github.com/yu-gopass/base/proto"
	"github.com/yu-gopass/common"
	"net"
	"net/http"

	"github.com/asim/go-micro/plugins/registry/consul/v3"
	opentracing2 "github.com/asim/go-micro/plugins/wrapper/trace/opentracing/v3"
	"github.com/asim/go-micro/v3/registry"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

func main() {

	//1.注册中心
	consul := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{
			"192.168.3.180:8500",
		}
	})

	//2.配置中心，存放经常使用的变量
	consulConfig, err := common.GetConsulConfig("192.168.3.180", 8500, "/micro/config")
	if err != nil {
		common.Error(err)
	}
	//3.使用配置中心连接mysql
	mysqlInfo := common.GetMysqlFromConsul(consulConfig, "mysql")
	//初始化mysql
	db, err := gorm.Open("mysql", mysqlInfo.User+":"+mysqlInfo.Pwd+"@tcp("+mysqlInfo.Host+":"+mysqlInfo.Port+")/"+mysqlInfo.Database+"?charset=utf8&parseTime&loc=Local")
	if err != nil {
		common.Fatal(err)
	}
	common.Info("连接mysql 成功")
	defer db.Close()
	//禁止重复表
	db.SingularTable(true)

	//4.添加链路追踪
	t, io, err := common.NewTracer("base", "192.168.3.180:6831")
	if err != nil {
		common.Error(err)
	}
	defer io.Close()
	opentracing.SetGlobalTracer(t)

	//5.添加熔断器
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	//启动监听程序
	go func() {
		//看板地址：http://192.168.3.180:9002/hystrix
		//程序监听地址：http://192.168.6.219:9092/turbine/turbine.stream
		err = http.ListenAndServe(net.JoinHostPort("0.0.0.0", "9092"), hystrixStreamHandler)
		if err != nil {
			common.Error(err)
		}
	}()

	//6.添加日志中心
	//1) 需要程序日志打入到日志文件中
	//2) 在程序中添加filebeat.yml文件
	//3) 启动filebeat, 启动命令 ./filebeat -e -c filebeat.yml
	common.Info("添加日志系统！")

	//7.添加监控
	common.PrometheusBoot(9192)

	// 创建服务
	service := micro.NewService(
		micro.Name("base"),
		micro.Version("latest"),
		//添加注册中心
		micro.Registry(consul),
		//添加链路追踪
		micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())),
		micro.WrapClient(opentracing2.NewClientWrapper(opentracing.GlobalTracer())),
		//作为客户端的时候启作用
		micro.WrapClient(hystrix2.NewClientHystrixWrapper()),
		//添加限流
		micro.WrapHandler(ratelimit.NewHandlerWrapper(1000)),
	)

	// 初始化服务
	service.Init()

	// 注册句柄，可以快速操作已开发的服务
	base.RegisterBaseHandler(service.Server(), new(handler.Base))

	// 运行服务
	if err := service.Run(); err != nil {
		common.Fatal(err)
	}
}
