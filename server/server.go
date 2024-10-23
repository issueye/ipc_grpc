package server

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/issueye/ipc_grpc/grpc/pb" // 替换为你的 proto 生成代码路径
	"github.com/issueye/ipc_grpc/vars"

	"github.com/Microsoft/go-winio"
	"google.golang.org/grpc"
)

type CommonHelpServer struct{}

// 测试网络
func (s *CommonHelpServer) Ping(context.Context, *pb.Empty) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "pong", Timestamp: time.Now().Unix()}, nil
}

// 获取插件信息
func (s *CommonHelpServer) SendInfo(context.Context, *pb.InfoRequest) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

// 心跳检测
func (s *CommonHelpServer) Heartbeat(stream pb.CommonHelper_HeartbeatServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break // 客户端已关闭流
		}

		if err != nil {
			fmt.Println("stream.Recv() failed: ", err.Error())
			return err
		}

		fmt.Println(req.MemoryUsage, req.CpuUsage, req.Message, req.Timestamp)
	}

	return nil
}

func NewServer(hostHelper pb.HostHelperServer) error {
	lis, err := winio.ListenPipe(vars.GetPipeName(), nil)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterCommonHelperServer(s, &CommonHelpServer{})
	pb.RegisterHostHelperServer(s, hostHelper)

	fmt.Println("grpc server start")
	if err := s.Serve(lis); err != nil {
		return nil
	}

	return nil
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
