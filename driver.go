package pkg

import (
	"context"
	"fmt"

	"github.com/edenzhang2012/storagequotainterface/sqi"
	"github.com/edenzhang2012/storagequotainterface/sqi/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type StorageQuotaPluginService struct {
	sqi.DefaultQuotaServiceServer
}

func NewStorageQuotaPluginService() *StorageQuotaPluginService {
	return &StorageQuotaPluginService{}
}

// 插件信息
func (s *StorageQuotaPluginService) GetPluginInfo(ctx context.Context, in *pb.PluginInfoRequest) (*pb.PluginInfoResponse, error) {
	return &pb.PluginInfoResponse{
		Name:          "gemini-sqi-driver test",
		VendorVersion: "1.0.0",
	}, nil
}

// 插件能力
func (s *StorageQuotaPluginService) GetPluginCapabilities(ctx context.Context, in *pb.GetPluginCapabilitiesRequest) (*pb.GetPluginCapabilitiesResponse, error) {

	return &pb.GetPluginCapabilitiesResponse{
		Capabilities: []*pb.PluginCapability{
			////////////////// quota 能力 //////////////////
			//可以设置Quota
			{Capability: &pb.PluginCapability_Rpc{Rpc: pb.PluginCapability_SET_QUOTA}},
			//可以清除Quota
			{Capability: &pb.PluginCapability_Rpc{Rpc: pb.PluginCapability_CLEAR_QUOTA}},
			//可以获取具体quota的基本信息
			{Capability: &pb.PluginCapability_Rpc{Rpc: pb.PluginCapability_GET_QUOTA}},
			//可以批量获取quota信息
			{Capability: &pb.PluginCapability_Rpc{Rpc: pb.PluginCapability_LIST_QUOTA}},
			//可以校验Quota
			{Capability: &pb.PluginCapability_Rpc{Rpc: pb.PluginCapability_VALIDATE_QUOTA}},
			////////////////// quota 类型： size or inode //////////////////
			//quota 类型:size
			{Capability: &pb.PluginCapability_Quota{Quota: pb.PluginCapability_SIZE}},
			////////////////// quota 唯一标识： path or id //////////////////
			//quota 唯一标识:path
			{Capability: &pb.PluginCapability_Id{Id: pb.PluginCapability_PATH}},
		},
	}, nil
}

// 设置配额
func (s *StorageQuotaPluginService) SetQuota(ctx context.Context, in *pb.SetQuotaRequest) (*pb.SetQuotaResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// 获取配额和用量
func (s *StorageQuotaPluginService) GetQuota(ctx context.Context, in *pb.GetQuotaRequest) (*pb.GetQuotaResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// 清理配额
func (s *StorageQuotaPluginService) ClearQuota(ctx context.Context, in *pb.ClearQuotaRequest) (*emptypb.Empty, error) {
	return nil, fmt.Errorf("not implemented")
}

// 查看所有配额
func (s *StorageQuotaPluginService) ListQuotas(ctx context.Context, in *pb.ListQuotasRequest) (*pb.ListQuotasResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// Optional
// 校验设置配额请求是否正确
func (s *StorageQuotaPluginService) ValidateQuotaRequest(ctx context.Context, in *pb.SetQuotaRequest) (*emptypb.Empty, error) {
	return nil, fmt.Errorf("not implemented")
}
