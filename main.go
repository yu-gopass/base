package main

import (
	"fmt"
	"github.com/asim/go-micro/v3"
	"github.com/yu-gopass/base/handler"
	base "github.com/yu-gopass/base/proto"
	"github.com/yu-gopass/common"
	"log"

	"github.com/asim/go-micro/plugins/registry/consul/v3"
	"github.com/asim/go-micro/v3/registry"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/micro/micro/v3/service/logger"
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
		fmt.Println(err)
	}
	//3.使用配置中心连接mysql
	mysqlInfo := common.GetMysqlFromConsul(consulConfig, "mysql")
	//初始化mysql
	db, err := gorm.Open("mysql", mysqlInfo.User+":"+mysqlInfo.Pwd+"@tcp("+mysqlInfo.Host+":"+mysqlInfo.Port+")/"+mysqlInfo.Database+"?charset=utf8&parseTime&loc=Local")
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	fmt.Println("连接mysql 成功")
	defer db.Close()
	//禁止重复表
	db.SingularTable(true)

	// 创建服务
	service := micro.NewService(
		micro.Name("base"),
		micro.Version("latest"),
		micro.Registry(consul),
	)

	// 初始化服务
	service.Init()

	// 注册句柄，可以快速操作已开发的服务
	base.RegisterBaseHandler(service.Server(), new(handler.Base))

	// 运行服务
	if err := service.Run(); err != nil {
		logger.Fatal(err)
	}
}
