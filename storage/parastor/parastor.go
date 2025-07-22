package parastor

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/edenzhang2012/geminisqidriver/storage"
	"github.com/edenzhang2012/geminisqidriver/tools"
	"github.com/edenzhang2012/storagequotainterface/sqi/pb"
)

const (
	LoginURL    = "/restLogin"
	GetNodeNum  = "/node/total"
	QuotaAdd    = "/quota/add"
	QuotaGet    = "/quota/info"
	QuotaList   = "/quota/list"
	QuotaDelete = "/quota/delete"
)

type ParastorQuotaServer struct {
	Ip       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`

	filesystemName string
	rootPath       string

	quotaHttpClinet *tools.APIClient

	lock  sync.Mutex
	token string
}

func newParastorQuotaServer(ip, username, password string, port int) (storage.Storage, error) {
	parastorQuotaServer := ParastorQuotaServer{
		Ip:       ip,
		Port:     port,
		Username: username,
		Password: password,

		filesystemName: tools.Config.FilesystemName,
		rootPath:       tools.Config.RootPath,
	}

	//init quota http client
	baseUrl := fmt.Sprintf("http://%s:%d", parastorQuotaServer.Ip, parastorQuotaServer.Port)
	apiClient, err := tools.NewAPIClient(baseUrl)
	if err != nil {
		return nil, err
	}
	parastorQuotaServer.quotaHttpClinet = apiClient

	//get token
	token, err := parastorQuotaServer.getToken()
	if err != nil {
		return nil, err
	}
	parastorQuotaServer.lock.Lock()
	parastorQuotaServer.token = token
	parastorQuotaServer.lock.Unlock()

	//refresh token
	go func() {
		maxRetry := 30 //最大重试30次，重试时间30秒
		retry := 0
		for retry < maxRetry {
			if _, err := parastorQuotaServer.getNode(); err != nil {
				retry++
				time.Sleep(time.Second)
			} else {
				retry = 0
				time.Sleep(5 * time.Minute) //5分钟刷新一次
			}
		}
	}()

	return &parastorQuotaServer, nil
}

func (qs *ParastorQuotaServer) getToken() (string, error) {
	params := url.Values{}
	params.Add("username", qs.Username)
	params.Add("password", qs.Password)
	params.Add("clientType", "REST")

	var res = struct {
		DetailErrMsg   string `json:"detail_ err_ msg"`
		ErrMsg         string `json:"err_msg"`
		ErrNo          int    `json:"err_no"`
		Sync           bool   `json:"sync"`
		TimeStamp      int64  `json:"time_stamp"`
		TimeZoneOffset int64  `json:"time_ zone_ offset"`
	}{}

	opts := tools.RequestOptions{
		Method:         "POST",
		Path:           LoginURL,
		Headers:        nil,
		Query:          params,
		Body:           nil,
		ResponseTarget: &res,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	httpRes, err := qs.quotaHttpClinet.Do(ctx, &opts)
	if err != nil {
		return "", err
	}
	if res.ErrNo != 200 {
		return "", fmt.Errorf("%s", res.ErrMsg)
	}
	token, ok := httpRes.Header[http.CanonicalHeaderKey("token")]
	if !ok {
		return "", fmt.Errorf("token is empty")
	}
	if token[0] == "" {
		return "", fmt.Errorf("token is empty")
	}
	return token[0], nil
}

// 获取节点数量，曙光存储中该接口消耗资源非常少，适合用来刷新token
func (qs *ParastorQuotaServer) getNode() (int, error) {
	var res = struct {
		DetailErrMsg   string `json:"detail_ err_ msg"`
		ErrMsg         string `json:"err_msg"`
		ErrNo          int    `json:"err_no"`
		Sync           bool   `json:"sync"`
		TimeStamp      int64  `json:"time_stamp"`
		TimeZoneOffset int64  `json:"time_ zone_ offset"`
		TraceId        string `json:"trace_id"`

		Result struct {
			NodeTotal int `json:"node_total"`
		} `json:"result"`
	}{}

	opts := tools.RequestOptions{
		Method:         "GET",
		Path:           GetNodeNum,
		Headers:        map[string]string{"token": qs.token},
		Query:          nil,
		Body:           nil,
		ResponseTarget: &res,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := qs.quotaHttpClinet.Do(ctx, &opts)
	if err != nil {
		return 0, err
	}
	if res.ErrNo != 200 {
		return 0, fmt.Errorf("%s", res.ErrMsg)
	}

	if res.Result.NodeTotal <= 0 {
		return 0, fmt.Errorf("get node failed")
	}

	return res.Result.NodeTotal, nil
}

// 插件信息，根据后端存储填写
func (qs *ParastorQuotaServer) GetPluginInfo(in *pb.PluginInfoRequest) (*pb.PluginInfoResponse, error) {
	return &pb.PluginInfoResponse{
		Name:          "parastor",
		VendorVersion: "v1",
	}, nil
}

// 插件能力，根据后端存储实际拥有的能力填写
func (qs *ParastorQuotaServer) GetPluginCapabilities(in *pb.GetPluginCapabilitiesRequest) (*pb.GetPluginCapabilitiesResponse, error) {

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
func (qs *ParastorQuotaServer) SetQuota(in *pb.SetQuotaRequest) (*pb.SetQuotaResponse, error) {
	//parastor 目前只使用目录配额
	if in.Target.Scope != pb.QuotaTarget_PATH {
		return nil, fmt.Errorf("unsupported target scope %d", in.Target.Scope)
	}

	var res = struct {
		DetailErrMsg   string `json:"detail_ err_ msg"`
		ErrMsg         string `json:"err_msg"`
		ErrNo          int    `json:"err_no"`
		Sync           bool   `json:"sync"`
		TimeStamp      int64  `json:"time_stamp"`
		TimeZoneOffset int64  `json:"time_ zone_ offset"`
		TraceId        string `json:"trace_id"`
	}{}

	params := url.Values{}
	path := fmt.Sprintf("%s:%s%s", qs.filesystemName, qs.rootPath, in.Target.Id)
	params.Add("path", path)                                                   //限制的路径
	params.Add("quota_type", "DIR QUOTA")                                      //配额类型为目录配额
	params.Add("logical_quota_cal_type", "QUOTA_LIMIT")                        //计算类型为限制，即达到使用量即不再允许写入
	params.Add("logical_hard_threshold", strconv.FormatUint(in.SizeBytes, 10)) //设置配额大小
	opts := tools.RequestOptions{
		Method:         "POST",
		Path:           QuotaAdd,
		Headers:        map[string]string{"token": qs.token},
		Query:          params,
		Body:           nil,
		ResponseTarget: &res,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := qs.quotaHttpClinet.Do(ctx, &opts)
	if err != nil {
		return nil, err
	}
	if res.ErrNo != 200 {
		return nil, fmt.Errorf("%s", res.ErrMsg)
	}

	return &pb.SetQuotaResponse{}, nil
}

// 获取配额和用量
func (qs *ParastorQuotaServer) GetQuota(in *pb.GetQuotaRequest) (*pb.GetQuotaResponse, error) {
	//parastor 目前只使用目录配额
	if in.Target.Scope != pb.QuotaTarget_PATH {
		return nil, fmt.Errorf("unsupported target scope %d", in.Target.Scope)
	}

	var res = struct {
		DetailErrMsg   string `json:"detail_ err_ msg"`
		ErrMsg         string `json:"err_msg"`
		ErrNo          int    `json:"err_no"`
		Sync           bool   `json:"sync"`
		TimeStamp      int64  `json:"time_stamp"`
		TimeZoneOffset int64  `json:"time_ zone_ offset"`
		TraceId        string `json:"trace_id"`

		//只取需要的字段
		Result struct {
			Limit  int `json:"limit"`
			Quotas []struct {
				// AuthProviderId                int    `json:"auth_provider_id"`
				// AuthProviderName              string `json:"auth_provider_name"`
				// CanBeDeletedInWebui           bool   `json:"can_be_deleted_in_webui"`
				// DefaultQuotaId                int    `json:"default_quota_id"`
				// Description                   string `json:"description"`
				// FilenrGraceTime               int    `json:"filenr_grace_time"`
				// FilenrHardThreshold           int64    `json:"filenr_hard_threshold"`
				// FilenrQuotaCalType            string `json:"filenr_quota_cal_type"`
				// FilenrSoftThreshold           int64    `json:"filenr_soft_threshold"`
				// FilenrSoftThresholdOverTime   string `json:"filenr_soft_threshold_over_time"`
				// FilenrSuggestThreshold        int64    `json:"filenr_suggest_threshold"`
				// FilenrUsedDirNr               int64    `json:"filenr_used_dir_nr"`
				// FilenrUsedNr                  int64    `json:"filenr_used_nr"`
				// FromServiceType               int    `json:"from_service_type"`
				// Fsid                          int    `json:"fsid"`
				// Id                            int    `json:"id"`
				// Idesc                         string `json:"idesc"`
				// IopsQuota                     int    `json:"iops_quota"`
				// IopsReal                      int    `json:"iops_real"`
				// IpsQuota                      int    `json:"ips_quota"`
				// IpsReal                       int    `json:"ips_real"`
				// Key                           int    `json:"key"`
				// LogicalGraceTime              int    `json:"logical_grace_time"`
				LogicalHardThreshold uint64 `json:"logical_hard_threshold"`
				// LogicalHardThresholdUnit      string `json:"logical_hard_threshold_unit"`
				LogicalQuotaCalType string `json:"logical_quota_cal_type"`
				// LogicalSoftThreshold          int64    `json:"logical_soft_threshold"`
				// LogicalSoftThresholdOverTime  string `json:"logical_soft_threshold_over_time"`
				// LogicalSoftThresholdUnit      string `json:"logical_soft_threshold_unit"`
				// LogicalSuggestThreshold       int64    `json:"logical_suggest_threshold"`
				// LogicalSuggestThresholdUnit   string `json:"logical_suggest_threshold_unit"`
				LogicalUsedCapacity uint64 `json:"logical_used_capacity"`
				// MetaIopsQuota                 int    `json:"meta_iops_quota"`
				// MetaIopsReal                  int    `json:"meta_iops_real"`
				// OpsQuota                      int    `json:"ops_quota"`
				// OpsReal                       int    `json:"ops_real"`
				Path string `json:"path"`
				// PhysicalCountRedundantSpace   bool   `json:"physical_count_redundant_space"`
				// PhysicalCountSnapshot         bool   `json:"physical_count_snapshot"`
				// PhysicalGraceTime             int    `json:"physical_grace_time"`
				// PhysicalHardThreshold         int    `json:"physical_hard_threshold"`
				// PhysicalHardThresholdUnit     string `json:"physical_hard_threshold_unit"`
				// PhysicalQuotaCalType          string `json:"physical_quota_cal_type"`
				// PhysicalSoftThreshold         int    `json:"physical_soft_threshold"`
				// PhysicalSoftThresholdOverTime string `json:"physical_soft_threshold_over_time"`
				// PhysicalSoftThresholdUnit     string `json:"physical_soft_threshold_unit"`
				// PhysicalSuggestThreshold      int    `json:"physical_suggest_threshold"`
				// PhysicalSuggestThresholdUnit  string `json:"physical_suggest_threshold_unit"`
				// PhysicalUsedCapacity          int    `json:"physical_used_capacity"`
				// ProductServiceType            string `json:"product_service_type"`
				QuotaType string `json:"quota_type"`
				// QuotaUpdateMode               string `json:"quota_update_mode"`
				// ReadBandwidthQuota            int    `json:"read_bandwidth_quota"`
				// ReadBandwidthReal             int    `json:"read_bandwidth_real"`
				// ReadWriteBandwidthQuota       int    `json:"read_write_bandwidth_quota"`
				// ReadWriteBandwidthReal        int    `json:"read_write_bandwidth_real"`
				State string `json:"state"`
				// TotalCounted                  bool   `json:"total_counted"`
				// UserOrGroupId                 int    `json:"user_or_group_id"`
				// UserOrGroupName               string `json:"user_or_group_name"`
				// UserType                      string `json:"user_type"`
				// Version                       int    `json:"version"`
				// WriteBandwidthQuota           int    `json:"write_bandwidth_quota"`
				// WriteBandwidthReal            int    `json:"write_bandwidth_real"`
			} `json:"quotas"`
			Searches []interface{} `json:"searches"`
			Sort     string        `json:"sort"`
			Start    int           `json:"start"`
			Total    int           `json:"total"`
		} `json:"result"`
	}{}

	type QuotaOperate struct {
		AbsolutePath string `json:"absolute_path"`
		Gid          int    `json:"gid"`
		Uid          int    `json:"uid"`
	}
	path := fmt.Sprintf("%s:%s%s", qs.filesystemName, qs.rootPath, in.Target.Id)
	var body = struct {
		QuotaOperateViews []QuotaOperate `json:"quota_operate_views"`
	}{
		QuotaOperateViews: []QuotaOperate{
			{
				AbsolutePath: path,
			},
		},
	}

	opts := tools.RequestOptions{
		Method:         "POST",
		Path:           QuotaGet,
		Headers:        map[string]string{"token": qs.token},
		Query:          nil,
		Body:           &body,
		ResponseTarget: &res,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := qs.quotaHttpClinet.Do(ctx, &opts)
	if err != nil {
		return nil, err
	}
	if res.ErrNo != 200 {
		return nil, fmt.Errorf("%s", res.ErrMsg)
	}
	if len(res.Result.Quotas) <= 0 {
		return nil, fmt.Errorf("get nothing from backend storage server")
	}

	quotaInfo := res.Result.Quotas[0]
	//check respanse
	if quotaInfo.Path == "" || quotaInfo.Path != path {
		return nil, fmt.Errorf("get path %s expected %s", quotaInfo.Path, path)
	}

	return &pb.GetQuotaResponse{
		Entry: &pb.QuotaEntry{
			Target: &pb.QuotaTarget{
				Scope: pb.QuotaTarget_PATH,
				Id:    path,
			},
			SizeBytes:        quotaInfo.LogicalHardThreshold,
			UsedBytes:        quotaInfo.LogicalUsedCapacity,
			InodeQuotaEnable: false,
		},
	}, nil
}

// 清理配额
func (qs *ParastorQuotaServer) ClearQuota(in *pb.ClearQuotaRequest) error {
	//parastor 目前只使用目录配额
	if in.Target.Scope != pb.QuotaTarget_PATH {
		return fmt.Errorf("unsupported target scope %d", in.Target.Scope)
	}

	var res = struct {
		DetailErrMsg   string `json:"detail_ err_ msg"`
		ErrMsg         string `json:"err_msg"`
		ErrNo          int    `json:"err_no"`
		Sync           bool   `json:"sync"`
		TimeStamp      int64  `json:"time_stamp"`
		TimeZoneOffset int64  `json:"time_ zone_ offset"`
		TraceId        string `json:"trace_id"`
	}{}

	type QuotaOperate struct {
		AbsolutePath string `json:"absolute_path"`
		Gid          int    `json:"gid"`
		Uid          int    `json:"uid"`
	}
	path := fmt.Sprintf("%s:%s%s", qs.filesystemName, qs.rootPath, in.Target.Id)
	var body = struct {
		QuotaOperateViews []QuotaOperate `json:"quota_operate_views"`
	}{
		QuotaOperateViews: []QuotaOperate{
			{
				AbsolutePath: path,
			},
		},
	}
	opts := tools.RequestOptions{
		Method:         "DELETE",
		Path:           QuotaDelete,
		Headers:        map[string]string{"token": qs.token},
		Query:          nil,
		Body:           body,
		ResponseTarget: &res,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := qs.quotaHttpClinet.Do(ctx, &opts)
	if err != nil {
		return err
	}
	if res.ErrNo != 200 {
		return fmt.Errorf("%s", res.ErrMsg)
	}

	return nil
}

// 查看所有配额
func (qs *ParastorQuotaServer) ListQuotas(in *pb.ListQuotasRequest) (*pb.ListQuotasResponse, error) {
	//parastor 目前只使用目录配额
	if in.Target.Scope != pb.QuotaTarget_PATH {
		return nil, fmt.Errorf("unsupported target scope %d", in.Target.Scope)
	}

	var res = struct {
		DetailErrMsg   string `json:"detail_ err_ msg"`
		ErrMsg         string `json:"err_msg"`
		ErrNo          int    `json:"err_no"`
		Sync           bool   `json:"sync"`
		TimeStamp      int64  `json:"time_stamp"`
		TimeZoneOffset int64  `json:"time_ zone_ offset"`
		TraceId        string `json:"trace_id"`

		//只取需要的字段
		Result struct {
			Limit  int `json:"limit"`
			Quotas []struct {
				// AuthProviderId                int    `json:"auth_provider_id"`
				// AuthProviderName              string `json:"auth_provider_name"`
				// CanBeDeletedInWebui           bool   `json:"can_be_deleted_in_webui"`
				// DefaultQuotaId                int    `json:"default_quota_id"`
				// Description                   string `json:"description"`
				// FilenrGraceTime               int    `json:"filenr_grace_time"`
				// FilenrHardThreshold           int64    `json:"filenr_hard_threshold"`
				// FilenrQuotaCalType            string `json:"filenr_quota_cal_type"`
				// FilenrSoftThreshold           int64    `json:"filenr_soft_threshold"`
				// FilenrSoftThresholdOverTime   string `json:"filenr_soft_threshold_over_time"`
				// FilenrSuggestThreshold        int64    `json:"filenr_suggest_threshold"`
				// FilenrUsedDirNr               int64    `json:"filenr_used_dir_nr"`
				// FilenrUsedNr                  int64    `json:"filenr_used_nr"`
				// FromServiceType               int    `json:"from_service_type"`
				// Fsid                          int    `json:"fsid"`
				// Id                            int    `json:"id"`
				// Idesc                         string `json:"idesc"`
				// IopsQuota                     int    `json:"iops_quota"`
				// IopsReal                      int    `json:"iops_real"`
				// IpsQuota                      int    `json:"ips_quota"`
				// IpsReal                       int    `json:"ips_real"`
				// Key                           int    `json:"key"`
				// LogicalGraceTime              int    `json:"logical_grace_time"`
				LogicalHardThreshold uint64 `json:"logical_hard_threshold"`
				// LogicalHardThresholdUnit      string `json:"logical_hard_threshold_unit"`
				LogicalQuotaCalType string `json:"logical_quota_cal_type"`
				// LogicalSoftThreshold          int64    `json:"logical_soft_threshold"`
				// LogicalSoftThresholdOverTime  string `json:"logical_soft_threshold_over_time"`
				// LogicalSoftThresholdUnit      string `json:"logical_soft_threshold_unit"`
				// LogicalSuggestThreshold       int64    `json:"logical_suggest_threshold"`
				// LogicalSuggestThresholdUnit   string `json:"logical_suggest_threshold_unit"`
				LogicalUsedCapacity uint64 `json:"logical_used_capacity"`
				// MetaIopsQuota                 int    `json:"meta_iops_quota"`
				// MetaIopsReal                  int    `json:"meta_iops_real"`
				// OpsQuota                      int    `json:"ops_quota"`
				// OpsReal                       int    `json:"ops_real"`
				Path string `json:"path"`
				// PhysicalCountRedundantSpace   bool   `json:"physical_count_redundant_space"`
				// PhysicalCountSnapshot         bool   `json:"physical_count_snapshot"`
				// PhysicalGraceTime             int    `json:"physical_grace_time"`
				// PhysicalHardThreshold         int    `json:"physical_hard_threshold"`
				// PhysicalHardThresholdUnit     string `json:"physical_hard_threshold_unit"`
				// PhysicalQuotaCalType          string `json:"physical_quota_cal_type"`
				// PhysicalSoftThreshold         int    `json:"physical_soft_threshold"`
				// PhysicalSoftThresholdOverTime string `json:"physical_soft_threshold_over_time"`
				// PhysicalSoftThresholdUnit     string `json:"physical_soft_threshold_unit"`
				// PhysicalSuggestThreshold      int    `json:"physical_suggest_threshold"`
				// PhysicalSuggestThresholdUnit  string `json:"physical_suggest_threshold_unit"`
				// PhysicalUsedCapacity          int    `json:"physical_used_capacity"`
				// ProductServiceType            string `json:"product_service_type"`
				QuotaType string `json:"quota_type"`
				// QuotaUpdateMode               string `json:"quota_update_mode"`
				// ReadBandwidthQuota            int    `json:"read_bandwidth_quota"`
				// ReadBandwidthReal             int    `json:"read_bandwidth_real"`
				// ReadWriteBandwidthQuota       int    `json:"read_write_bandwidth_quota"`
				// ReadWriteBandwidthReal        int    `json:"read_write_bandwidth_real"`
				State string `json:"state"`
				// TotalCounted                  bool   `json:"total_counted"`
				// UserOrGroupId                 int    `json:"user_or_group_id"`
				// UserOrGroupName               string `json:"user_or_group_name"`
				// UserType                      string `json:"user_type"`
				// Version                       int    `json:"version"`
				// WriteBandwidthQuota           int    `json:"write_bandwidth_quota"`
				// WriteBandwidthReal            int    `json:"write_bandwidth_real"`
			} `json:"quotas"`
			Searches []interface{} `json:"searches"`
			Sort     string        `json:"sort"`
			Start    int           `json:"start"`
			Total    int           `json:"total"`
		} `json:"result"`
	}{}

	//TODO
	// type QuotaOperate struct {
	// 	AbsolutePath string `json:"absolute_path"`
	// 	Gid          int    `json:"gid"`
	// 	Uid          int    `json:"uid"`
	// }
	path := fmt.Sprintf("%s:%s%s", qs.filesystemName, qs.rootPath, in.Target.Id)
	// var body = struct {
	// 	QuotaOperateViews []QuotaOperate `json:"quota_operate_views"`
	// }{
	// 	QuotaOperateViews: []QuotaOperate{
	// 		{
	// 			AbsolutePath: path,
	// 		},
	// 	},
	// }

	opts := tools.RequestOptions{
		Method:         "POST",
		Path:           QuotaList,
		Headers:        map[string]string{"token": qs.token},
		Query:          nil,
		Body:           nil,
		ResponseTarget: &res,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := qs.quotaHttpClinet.Do(ctx, &opts)
	if err != nil {
		return nil, err
	}
	if res.ErrNo != 200 {
		return nil, fmt.Errorf("%s", res.ErrMsg)
	}
	if len(res.Result.Quotas) <= 0 {
		return nil, fmt.Errorf("get nothing from backend storage server")
	}

	quotaInfo := res.Result.Quotas[0]
	//check respanse
	if quotaInfo.Path == "" || quotaInfo.Path != path {
		return nil, fmt.Errorf("get path %s expected %s", quotaInfo.Path, path)
	}

	return &pb.ListQuotasResponse{}, nil
}

// Optional
// 校验设置配额请求是否正确
func (qs *ParastorQuotaServer) ValidateQuotaRequest(in *pb.SetQuotaRequest) error {
	return fmt.Errorf("not implemented")
}

func init() {
	storage.Register("parastor", newParastorQuotaServer)
}
