package zlog

import (
	"kama-chat-server/internal/config"
	"os"
	"path"
	"runtime"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)


var logger *zap.Logger
var logPath string


func init() {
	// 1.获取配置中的日志路径
	conf := config.GetConfig()
	logPath = conf.LogConfig.LogPath

	// 2.配置编码器
	encoderConfig := zap.NewProductionEncoderConfig()

	// 时间格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Json编码器
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// 3.创建多输出流
	//  同时输出到控制台和文件
	consoleWriteSyncer := zapcore.AddSync(os.Stdout)

	// 文件输出
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        panic("日志文件创建失败: " + err.Error())
    }
    fileWriteSyncer := zapcore.AddSync(file)

    // 4. 创建Core
    // ★zapcore.NewTee: 同时使用多个输出流
    core := zapcore.NewTee(
        zapcore.NewCore(encoder, consoleWriteSyncer, zapcore.DebugLevel), // 控制台
        zapcore.NewCore(encoder, fileWriteSyncer, zapcore.DebugLevel),    // 文件
    )

    // 5. 创建Logger
    logger = zap.New(core, zap.AddCaller())

}


// ============================================================
// getCallerInfoForLog - 获取调用信息
// ============================================================
// ★runtime.Caller(skip): 回溯调用栈
// skip=0: 当前函数
// skip=1: 调用当前函数的函数
// skip=2: 再上一级（实际的日志调用位置）

func getCallerInfoForLog() []zap.Field {
	pc, file, line, ok := runtime.Caller(2)  // ★回溯两层
    if !ok {
        return nil
    }

    // 获取函数名
    funcName := path.Base(runtime.FuncForPC(pc).Name())

    // 返回zap.Field数组
    return []zap.Field{
        zap.String("func", funcName),  // 函数名
        zap.String("file", file),      // 文件名
        zap.Int("line", line),         // 行号
    }
}


// Info 信息级别日志
func Info(message string, fields ...zap.Field) {
	fields = append(fields, getCallerInfoForLog()...)
	logger.Info(message, fields...)
}

// Error - 错误级别日志
func Error(message string, fields ...zap.Field) {
    fields = append(fields, getCallerInfoForLog()...)
    logger.Error(message, fields...)
}

// Warn - 警告级别日志
func Warn(message string, fields ...zap.Field) {
    fields = append(fields, getCallerInfoForLog()...)
    logger.Warn(message, fields...)
}
// Fatal - 致命错误日志（会终止程序）
func Fatal(message string, fields ...zap.Field) {
    fields = append(fields, getCallerInfoForLog()...)
    logger.Fatal(message, fields...)
}

// Debug - 调试级别日志
func Debug(message string, fields ...zap.Field) {
    fields = append(fields, getCallerInfoForLog()...)
    logger.Debug(message, fields...)
}
















