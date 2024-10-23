package vars

var (
	PIPE_BASE = `\\.\pipe\`
	PIPE_NAME = "ipc_grpc_pipe"
)

func GetPipeName() string {
	return PIPE_BASE + PIPE_NAME
}

var (
	Version     = "0.0.1"     // 版本号
	AppName     = "grpc-demo" // 应用名称
	GitHash     = ""          // git hash
	GitBranch   = ""          // git branch
	BuildTime   = ""          // 构建时间
	GoVersion   = ""          // go版本
	CookieKey   = ""          // cookie key
	CookieValue = ""          // cookie value
)
