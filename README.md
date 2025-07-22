# gemini sqi driver demo
这是sqi(storage quota interface)示例项目，该项目旨在构建一个适配Gemini AI训练平台的配额管理driver，其思路类似于CSI
# how to use
- 根据自己存储的配额能力，实现配额相关接口，接口如下:
```go
type QuotaServiceClient interface {
	// 插件信息
	GetPluginInfo(ctx context.Context, in *PluginInfoRequest, opts ...grpc.CallOption) (*PluginInfoResponse, error)
	// 插件能力
	GetPluginCapabilities(ctx context.Context, in *GetPluginCapabilitiesRequest, opts ...grpc.CallOption) (*GetPluginCapabilitiesResponse, error)
	// 设置配额
	SetQuota(ctx context.Context, in *SetQuotaRequest, opts ...grpc.CallOption) (*SetQuotaResponse, error)
	// 获取配额和用量
	GetQuota(ctx context.Context, in *GetQuotaRequest, opts ...grpc.CallOption) (*GetQuotaResponse, error)
	// 清理配额
	ClearQuota(ctx context.Context, in *ClearQuotaRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// 查看所有配额
	ListQuotas(ctx context.Context, in *ListQuotasRequest, opts ...grpc.CallOption) (*ListQuotasResponse, error)
	// Optional
	// 校验设置配额请求是否正确
	ValidateQuotaRequest(ctx context.Context, in *SetQuotaRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}
```
- 编译镜像
    -  依赖
        需要提前安装好golang和docker
    - 编译
        项目根目录执行`make`
- 将生成的镜像推送到gemini环境，使用build/server/k8s-storage-quota-plugin-daemonset.yaml部署到环境中
- 修改Gemini配置，重新部署相应的组件即可使用配额能力
# test
本项目同时也提供了sqi driver的client用于测试driver的功能是否正常，并且该测试不需要Gemini平台。使用方法如下:
- 执行`make`时会同时生成sqi driver server和client的二进制文件和docker镜像
- 将server和client镜像推送到k8s的镜像仓库中
- 使用`build/server/k8s-storage-quota-plugin-daemonset.yaml`部署server
- 使用`build/client/k8s-storage-quota-plugin-pod.yaml`部署client
- 观察client的日志，查看driver的功能是否正常