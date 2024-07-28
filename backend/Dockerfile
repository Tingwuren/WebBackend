# 使用官方的Go镜像作为基础镜像
FROM golang:1.22.4-alpine3.20


# 安装必要的构建工具和 SQLite3 库
RUN apk add --no-cache gcc musl-dev sqlite-dev

# 设置工作目录
WORKDIR /app

# 设置 Go Proxy
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=1

# 复制 go.mod 和 go.sum 文件到工作目录
COPY go.mod go.sum ./

# 下载依赖项
RUN go mod download

# 复制当前目录内容到工作目录
COPY . .

# 显示当前工作目录和文件
RUN pwd && ls -al

# 构建应用程序
RUN go build -v -o main .

# 暴露应用程序端口
EXPOSE 8080

# 启动应用程序
CMD ["./main"]

