package client

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "github.com/issueye/ipc_grpc/grpc/pb" // 替换为你的 proto 生成代码路径
	"github.com/issueye/ipc_grpc/vars"

	"github.com/Microsoft/go-winio"
	"google.golang.org/grpc"
)

type Client struct {
	commonHelper      pb.CommonHelperClient
	hostHelper        pb.HostHelperClient
	grpcConn          *grpc.ClientConn
	timeOut           time.Duration
	heartbeatInterval time.Duration

	stream pb.CommonHelper_HeartbeatClient
}

func NewClient() (*Client, error) {
	client := &Client{}

	// 创建 gRPC 连接
	grpcConn, err := grpc.Dial(
		"",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			timeOut := time.Second * 10
			return winio.DialPipe(vars.GetPipeName(), &timeOut)
		}), grpc.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}

	client.grpcConn = grpcConn
	client.commonHelper = pb.NewCommonHelperClient(grpcConn)
	client.hostHelper = pb.NewHostHelperClient(grpcConn)
	return client, nil
}

func (c *Client) CommonHelper() pb.CommonHelperClient {
	return c.commonHelper
}

func (c *Client) HostHelper() pb.HostHelperClient {
	return c.hostHelper
}

func (c *Client) SetHeartbeatInterval(interval time.Duration) {
	c.heartbeatInterval = interval
}

func (c *Client) SetTimeOut(timeOut time.Duration) {
	c.timeOut = timeOut
}

func (c *Client) GetTimeOut() time.Duration {
	if c.timeOut == 0 {
		c.timeOut = time.Second * 10
	}

	return c.timeOut
}

func (c *Client) Close() {
	c.grpcConn.Close()
}

func (c *Client) GetConn() *grpc.ClientConn {
	return c.grpcConn
}

func (c *Client) Ping() (*pb.PingResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeOut())
	defer cancel()

	return c.commonHelper.Ping(ctx, &pb.Empty{})
}

func (c *Client) SendInfo() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeOut())
	defer cancel()

	_, err := c.commonHelper.SendInfo(ctx, &pb.InfoRequest{
		Version:     vars.Version,     // 版本号
		AppName:     vars.AppName,     // 应用名称
		GitHash:     vars.GitHash,     // git hash
		GitBranch:   vars.GitBranch,   // git branch
		BuildTime:   vars.BuildTime,   // 构建时间
		GoVersion:   vars.GoVersion,   // go版本
		CookieKey:   vars.CookieKey,   // cookie key
		CookieValue: vars.CookieValue, // cookie value
	})

	return err
}

func (c *Client) Heartbeat() error {
	stream, err := c.commonHelper.Heartbeat(context.Background())
	if err != nil {
		return err
	}

	c.stream = stream

	// 允许失败3次
	i := 0

	go func() {
		interval := c.heartbeatInterval
		if interval == 0 {
			interval = time.Second * 10
		}

		ticker := time.NewTicker(interval)
		for range ticker.C {

			fmt.Println("开始发送心跳")

			// 获取CPU使用率
			// 获取内存使用率
			err := stream.Send(&pb.HeartbeatRequest{
				Message:     "ping",
				Timestamp:   time.Now().Unix(),
				MemoryUsage: 0,
				CpuUsage:    0,
			})

			if err != nil {
				i++
				if i >= 3 {
					ticker.Stop()
				}

				fmt.Println("心跳发送失败", err.Error())
				continue
			}

			fmt.Println("心跳发送成功")

			// 如果发送成功，则重置计数器
			i = 0
		}

		fmt.Println("心跳发送结束")
	}()

	return nil
}
