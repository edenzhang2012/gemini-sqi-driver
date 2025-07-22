package storage

import (
	"fmt"

	"github.com/edenzhang2012/storagequotainterface/sqi/pb"
)

type Storage interface {
	GetPluginInfo(*pb.PluginInfoRequest) (*pb.PluginInfoResponse, error)
	GetPluginCapabilities(*pb.GetPluginCapabilitiesRequest) (*pb.GetPluginCapabilitiesResponse, error)
	SetQuota(*pb.SetQuotaRequest) (*pb.SetQuotaResponse, error)
	GetQuota(*pb.GetQuotaRequest) (*pb.GetQuotaResponse, error)
	ClearQuota(*pb.ClearQuotaRequest) error
	ListQuotas(*pb.ListQuotasRequest) (*pb.ListQuotasResponse, error)
	ValidateQuotaRequest(in *pb.SetQuotaRequest) error
}

type Creator func(ip, username, password string, port int) (Storage, error)

var storages map[string]Creator

func NewStorage(name, ip, username, password string, port int) (Storage, error) {
	f, ok := storages[name]
	if ok {
		return f(ip, username, password, port)
	}

	return nil, fmt.Errorf("invalid storage %s", name)
}

func Register(name string, register Creator) {
	storages[name] = register
}
