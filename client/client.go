package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	pb "github.com/issueye/ipc_grpc/grpc/pb" // 替换为你的 proto 生成代码路径
	"github.com/issueye/ipc_grpc/vars"

	"github.com/Microsoft/go-winio"
	"github.com/shirou/gopsutil/v4/process"
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

	// 创建 gRPC 连接 (使用 Windows Named Pipe)
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

func (c *Client) Ping() (*pb.PubResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeOut())
	defer cancel()

	return c.commonHelper.Ping(ctx, &pb.Empty{})
}

func (c *Client) Register() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeOut())
	defer cancel()

	_, err := c.commonHelper.Register(ctx, &pb.InfoRequest{
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

	interval := c.heartbeatInterval
	if interval == 0 {
		interval = time.Second * 10
	}

	ticker := time.NewTicker(interval)
	for range ticker.C {
		info, err := GetClientState()
		if err != nil {
			fmt.Println("获取客户端状态失败", err.Error())
			continue
		}

		// 获取CPU使用率
		// 获取内存使用率
		err = stream.Send(&pb.HeartbeatRequest{
			CookieKey:   vars.CookieKey,
			Message:     "ping",
			Timestamp:   time.Now().Unix(),
			MemoryUsage: info.MemoryUsage,
			CpuUsage:    info.CpuUsage,
		})

		if err != nil {
			i++
			if i >= 3 {
				ticker.Stop()
			}

			continue
		}
		// 如果发送成功，则重置计数器
		i = 0
	}

	return nil
}

type StateInfo struct {
	Pid         int
	ProcessName string
	CpuUsage    float32
	MemoryUsage float32
}

func GetClientState() (*StateInfo, error) {
	pid := os.Getpid()

	list, err := process.Processes()
	if err != nil {
		return nil, err
	}

	info := &StateInfo{
		Pid: pid,
	}

	for _, p := range list {
		name, err := p.Name()
		if err != nil {
			continue
		}

		info.ProcessName = name

		if p.Pid == int32(pid) {
			cpu, err := p.CPUPercent()
			if err == nil {
				info.CpuUsage = float32(cpu)
			}

			// 获取内存使用率
			mem, err := p.MemoryInfo()
			if err == nil {
				info.MemoryUsage = float32(mem.RSS)
			}

			break
		}
	}

	return info, nil
}
