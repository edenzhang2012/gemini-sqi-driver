package pkg

import (
	"context"

	"github.com/edenzhang2012/geminisqidriver/storage"
	"github.com/edenzhang2012/geminisqidriver/tools"
	"github.com/edenzhang2012/storagequotainterface/sqi"
	"github.com/edenzhang2012/storagequotainterface/sqi/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type StorageQuotaPluginService struct {
	sqi.DefaultQuotaServiceServer

	storage storage.Storage
}

func NewStorageQuotaPluginService() (*StorageQuotaPluginService, error) {
	storage, err := storage.NewStorage(tools.Config.StorageName, tools.Config.Ip,
		tools.Config.Username, tools.Config.Password, tools.Config.Port)
	if err != nil {
		return nil, err
	}

	return &StorageQuotaPluginService{storage: storage}, nil
}

// 以下实现皆为透传
func (s *StorageQuotaPluginService) GetPluginInfo(ctx context.Context, in *pb.PluginInfoRequest) (*pb.PluginInfoResponse, error) {
	return s.storage.GetPluginInfo(in)
}
func (s *StorageQuotaPluginService) GetPluginCapabilities(ctx context.Context, in *pb.GetPluginCapabilitiesRequest) (*pb.GetPluginCapabilitiesResponse, error) {
	return s.storage.GetPluginCapabilities(in)
}
func (s *StorageQuotaPluginService) SetQuota(ctx context.Context, in *pb.SetQuotaRequest) (*pb.SetQuotaResponse, error) {
	return s.storage.SetQuota(in)
}
func (s *StorageQuotaPluginService) GetQuota(ctx context.Context, in *pb.GetQuotaRequest) (*pb.GetQuotaResponse, error) {
	return s.storage.GetQuota(in)
}
func (s *StorageQuotaPluginService) ClearQuota(ctx context.Context, in *pb.ClearQuotaRequest) (*emptypb.Empty, error) {
	err := s.storage.ClearQuota(in)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
func (s *StorageQuotaPluginService) ListQuotas(ctx context.Context, in *pb.ListQuotasRequest) (*pb.ListQuotasResponse, error) {
	return s.storage.ListQuotas(in)
}
func (s *StorageQuotaPluginService) ValidateQuotaRequest(ctx context.Context, in *pb.SetQuotaRequest) (*emptypb.Empty, error) {
	err := s.storage.ValidateQuotaRequest(in)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
