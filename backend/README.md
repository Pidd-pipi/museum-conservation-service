# Backend

Go 1.22 模块 museum-conservation-service，入口为 main.go，启动后提供 /health 健康检查。

## Standard Commands

在本目录执行：

    go test ./...
    go build ./...
    go run .

服务端口由 PORT 环境变量控制，默认 8080。backend/web.go 使用 go:embed 嵌入 backend/web/index.html 和 backend/web/app.js。

