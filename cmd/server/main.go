package main

import (
	"flag"
	"net"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/edenzhang2012/geminisqidriver/pkg"
	"github.com/edenzhang2012/geminisqidriver/tools"
	"github.com/edenzhang2012/storagequotainterface/sqi/pb"

	"google.golang.org/grpc"
)

var (
	Version   string
	BuildNo   string
	BuildTime string

	configFile = flag.String("config", "/etc/storagequotaplugin/config-example.yaml", "The path of the configFile")
)

const (
	AppName = "storagequotaplugin"
)

func init() {
	//log Init
	tools.LogInit()

	flag.Parse()
	if *configFile == "" {
		tools.Logger.Fatal("config is not setted!")
	}

	tools.ParseConfig(AppName, *configFile)
	tools.Logger.Infof("init config:%+v\n", *tools.Config)

	//get server info from config
	if tools.Config.Ip == "" || tools.Config.Ip == "None" ||
		tools.Config.Port == 0 ||
		tools.Config.Username == "" || tools.Config.Username == "None" ||
		tools.Config.Password == "" || tools.Config.Password == "None" {
		tools.Logger.Fatalf("quota server Ip、Port、Username、Password must be set")
	}
}

func main() {
	//pprof
	go func() {
		http.ListenAndServe("0.0.0.0:8899", nil)
	}()

	//delete old socket file
	os.Remove("/var/run/" + AppName + "/" + AppName + ".sock")

	//start gRPC server
	lis, err := net.Listen("unix", "/var/run/"+AppName+"/"+AppName+".sock")
	if err != nil {
		tools.Logger.Fatalf("failed to listen: %v", err)
	}

	sqp, err := pkg.NewStorageQuotaPluginService()
	if err != nil {
		tools.Logger.Errorf("NewStorageQuotaPluginService failed with %v", err)
		return
	}
	opts := []grpc.ServerOption{}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterQuotaServiceServer(grpcServer, sqp)

	if err := grpcServer.Serve(lis); err != nil {
		tools.Logger.Fatalf("grpc server err %v", err)
	}
}
