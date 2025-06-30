package main

import (
	"log"
	"net"
	"net/http"
	"os"

	_ "net/http/pprof"

	pkg "github.com/edenzhang2012/geminisqidriver"
	"github.com/edenzhang2012/storagequotainterface/sqi/pb"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

var (
	Version   string
	BuildNo   string
	BuildTime string

	logger *logrus.Logger
)

const (
	AppName = "storagequotaplugin"
)

func init() {
	//log config
	level := logrus.InfoLevel
	levelEnv := os.Getenv("LOG_LEVEL")
	if levelEnv != "" {
		var err error
		level, err = logrus.ParseLevel(levelEnv)
		if err != nil {
			log.Fatalf("Invalid log level: %s", levelEnv)
		}
	}

	logger = logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(level)
	logger.SetReportCaller(true)
}

func main() {
	//pprof
	go func() {
		http.ListenAndServe("0.0.0.0:8899", nil)
	}()

	//delete socket file
	os.Remove("/var/run/" + AppName + "/" + AppName + ".sock")

	//start gRPC server
	lis, err := net.Listen("unix", "/var/run/"+AppName+"/"+AppName+".sock")

	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}

	sqp := pkg.NewStorageQuotaPluginService()
	opts := []grpc.ServerOption{}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterQuotaServiceServer(grpcServer, sqp)

	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatalf("grpc server err %v", err)
	}
}
