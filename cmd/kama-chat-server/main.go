package main

import "kama-chat-server/pkg/zlog"

func main() {
    zlog.Info("服务器启动")
    zlog.Debug("调试信息")
    zlog.Warn("警告信息")
}