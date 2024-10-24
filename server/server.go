package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	pb "github.com/issueye/ipc_grpc/grpc/pb" // 替换为你的 proto 生成代码路径
	"github.com/issueye/ipc_grpc/vars"

	"github.com/Microsoft/go-winio"
	"google.golang.org/grpc"
)

type PluginInfo struct {
	Version           string `json:"version"`           // 版本号
	AppName           string `json:"appName"`           // 程序名称
	GitHash           string `json:"gitHash"`           // git hash
	GitBranch         string `json:"gitBranch"`         // git 分支
	BuildTime         string `json:"buildTime"`         // 构建时间
	GoVersion         string `json:"goVersion"`         // go 版本
	CookieKey         string `json:"cookieKey"`         // cookie key
	CookieValue       string `json:"cookieValue"`       // cookie value
	LastHeartbeatTime int64  `json:"lastHeartbeatTime"` // 最后心跳时间
	State             int32  `json:"state"`             // 状态
}

type PluginState struct {
	CookieKey         string `json:"cookieKey"`         // cookie key
	LastHeartbeatTime int64  `json:"lastHeartbeatTime"` // 最后心跳时间
	State             int32  `json:"state"`             // 状态
}

type CommonHelpServer struct {
	callback        func(info *PluginInfo) error
	updateHeartbeat func(cookieKey string, lastHeartbeatTime int64)
	states          map[string]*PluginState
}

func (s *CommonHelpServer) CheckPlugin(cookieKey string) (*PluginState, error) {
	state, ok := s.states[cookieKey]
	if !ok {
		return nil, fmt.Errorf("cookie key 不存在")
	}

	return state, nil
}

// 测试网络
func (s *CommonHelpServer) Ping(context.Context, *pb.Empty) (*pb.PubResponse, error) {
	return &pb.PubResponse{Message: "pong", Timestamp: time.Now().Unix()}, nil
}

// 获取插件信息
func (s *CommonHelpServer) Register(ctx context.Context, data *pb.InfoRequest) (*pb.PubResponse, error) {
	pi := &PluginInfo{
		Version:     data.Version,
		AppName:     data.AppName,
		GitHash:     data.GitHash,
		GitBranch:   data.GitBranch,
		BuildTime:   data.BuildTime,
		GoVersion:   data.GoVersion,
		CookieKey:   data.CookieKey,
		CookieValue: data.CookieValue,
		State:       1,
	}

	fmt.Printf("程序名称：%s，cookie key：%s，版本号：%s\n", pi.AppName, pi.CookieKey, pi.Version)
	if pi.CookieKey == "" {
		return nil, fmt.Errorf("cookie key 不能为空")
	}

	err := s.callback(pi)
	if err != nil {
		return nil, fmt.Errorf("插件注册失败: %s", err.Error())
	}

	s.states[pi.CookieKey] = &PluginState{
		CookieKey:         pi.CookieKey,
		LastHeartbeatTime: time.Now().Unix(),
		State:             1,
	}

	return &pb.PubResponse{Message: "ok", Timestamp: time.Now().Unix()}, nil
}

// 心跳检测
func (s *CommonHelpServer) Heartbeat(stream pb.CommonHelper_HeartbeatServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break // 客户端已关闭流
		}

		if err != nil {
			return stream.SendAndClose(&pb.PubResponse{Message: err.Error(), Timestamp: time.Now().Unix()})
		}

		if req.CookieKey == "" {
			return stream.SendAndClose(&pb.PubResponse{Message: "cookie key 不能为空", Timestamp: time.Now().Unix()})
		}

		state, err := s.CheckPlugin(req.CookieKey)
		if err != nil {
			return stream.SendAndClose(&pb.PubResponse{Message: err.Error(), Timestamp: time.Now().Unix()})
		}

		fmt.Printf("收到心跳包，cookie key：%s  内存使用：%.2f MB\n", req.CookieKey, req.MemoryUsage/1024.0/1024.0)
		state.LastHeartbeatTime = time.Now().Unix()
		state.State = 1

		s.updateHeartbeat(req.CookieKey, req.Timestamp)
	}

	return nil
}

func NewCommonHelpServer(callback func(*PluginInfo) error, updateHeartbeat func(cookieKey string, lastHeartbeatTime int64)) *CommonHelpServer {
	return &CommonHelpServer{
		callback:        callback,
		updateHeartbeat: updateHeartbeat,
		states:          make(map[string]*PluginState),
	}
}

type Server struct {
	Plugins          map[string]*PluginInfo
	hostHelper       pb.HostHelperServer
	commonHelpServer *CommonHelpServer
	checkPlugin      func(PluginInfo) error
}

func NewServer(isASync bool, hostHelper pb.HostHelperServer, checkPlugin func(PluginInfo) error) (server *Server, err error) {
	var (
		lis net.Listener
	)

	lis, err = winio.ListenPipe(vars.GetPipeName(), nil)
	if err != nil {
		return nil, err
	}

	s := grpc.NewServer()

	callBack := func(pi *PluginInfo) error {
		_, ok := server.Plugins[pi.CookieKey]
		if !ok {
			err := checkPlugin(*pi)
			if err != nil {
				return err
			}

			server.Plugins[pi.CookieKey] = pi
			return nil
		} else {
			return fmt.Errorf("cookie key 已存在")
		}
	}

	updateHeadbeat := func(cookieKey string, lastHeartbeatTime int64) {
		plugin, ok := server.Plugins[cookieKey]
		if !ok {
			return
		}

		plugin.LastHeartbeatTime = lastHeartbeatTime
	}

	commonHelpServer := NewCommonHelpServer(callBack, updateHeadbeat)
	server = &Server{
		Plugins:          make(map[string]*PluginInfo),
		hostHelper:       hostHelper,
		commonHelpServer: commonHelpServer,
		checkPlugin:      checkPlugin,
	}

	pb.RegisterCommonHelperServer(s, commonHelpServer)
	pb.RegisterHostHelperServer(s, hostHelper)

	fmt.Println("服务端启动...")

	if isASync {
		go func() {
			err = s.Serve(lis)
			if err != nil {
				fmt.Println("grpc server start failed: ", err.Error())
			}
		}()
	} else {
		err = s.Serve(lis)
		if err != nil {
			return nil, err
		}
	}

	return
}

type Host struct{}

// 服务状态
func (*Host) Status(context.Context, *pb.ServerRequest) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{}, nil
}

// 服务停止
func (*Host) Stop(context.Context, *pb.ServerRequest) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

// 服务启动
func (*Host) Start(context.Context, *pb.ServerRequest) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

// 服务重启
func (*Host) Restart(context.Context, *pb.ServerRequest) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

// 服务列表
func (*Host) List(context.Context, *pb.Empty) (*pb.ListResponse, error) {
	return &pb.ListResponse{}, nil
}

// 事件推送，服务端持续推送事件
func (*Host) Event(*pb.Empty, pb.HostHelper_EventServer) error {
	return nil
}
