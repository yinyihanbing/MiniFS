#build stage
#使用golang作为基础镜像
FROM golang:alpine AS builder
#--no-cache不使用缓存目录缓存，装完也不保留缓存
RUN apk add --no-cache git
#将工作目录切换到指定目录
WORKDIR /minifs
#拷贝当前工作目录文件到容器目录
COPY . .
#一键下载依赖 -d下载不安装 -v显示下载LOG ./...表示所有文件
RUN go get -d -v ./...
#执行go编译命令, 编译输出可执行文件app
RUN go build -o /minifs/app -v ./...

#final stage
#使用alpine作为基础镜像,该镜像为linux的一个发行版
FROM alpine:latest
#运行包管理工具apk安装ca-certificates软件包,软件包含一组根证书和中间证书,用SSL/TLS通信使用
RUN apk --no-cache add ca-certificates
#从上个阶段临时构建的镜像中拷贝文件或目录
COPY --from=builder /minifs/app /minifs/app
#容器启动后执行的命令, 运行可执行文件app
ENTRYPOINT /minifs/app
#镜像标签及版本号
LABEL Name=minifs Version=0.0.1
#导出的端口号
EXPOSE 8282