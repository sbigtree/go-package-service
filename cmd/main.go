package main

import (
	"context"
	"fmt"
	"github.com/sbigtree/go-package-service/cmd/global"
	_ "github.com/sbigtree/go-package-service/cmd/initialize"
	"github.com/sbigtree/go-package-service/core/mq"
	"github.com/sbigtree/go-package-service/core/scheduler/consumer"
	"github.com/sbigtree/go-package-service/heapdump"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {

	//读环境变量
	global.DEV = os.Getenv("DEV")

	defer func() {
		if r := recover(); r != nil {
			zap.S().Errorf("主 goroutine panic: %v\n堆栈信息:\n%s", r, debug.Stack())
			// 可以扩展
		}
	}()
	defer recoverMainPanic()
	addr := "0.0.0.0:" + strconv.Itoa(global.CMDConfig.Port)

	// 1. 监听
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		zap.S().Fatal("监听端口失败", zap.Error(err))
	}
	zap.S().Info("✅ 监听端口：", addr)

	// 2. 实例化 gRPC 服务器
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			RecoveryInterceptor, // 新增的 panic 拦截器
		),
	)
	// 3. 注册服务
	//tradeupBuf.RegisterCs2TradeupGrpcServer(s, &handle.Cs2TradeUpGrpcServer{})

	// 4. 启动 gRPC 服务
	go SafeGo(func() {
		zap.S().Info("✅ gRPC 服务器启动中...")
		if err := s.Serve(listener); err != nil {
			zap.S().Error("gRPC 服务启动失败", zap.Error(err))
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 启动内部消费者
	go func() {
		consumer.StartInternalConsumer(ctx)
	}()
	// 5. 启动 RocketMQ 消费者
	go SafeGo(func() {
		err := mq.StartRocketMQConsumer()
		if err != nil {
			zap.S().Errorf("mq消费者启动失败")
		} else {
			zap.S().Info("✅ 所有消费者启动完成")
		}

	})

	// 启动内存监控
	heapdump.StartHeapDump(heapdump.HeapDumpConfig{
		Dir:      "./tmp",         // Docker 内路径
		Interval: 2 * time.Minute, // 🔥 推荐 1~5 分钟
		//Interval: 10 * time.Second, // 🔥 推荐 1~5 分钟
		MaxFiles:    100, // 最多留 10 个
		Enable:      true,
		ThresholdMB: 1000,
	})

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	zap.S().Info("🔔 收到退出信号，正在关闭服务...")

	// 通知所有 goroutine 停止
	cancel()
	s.GracefulStop()
	zap.S().Info("服务已完全关闭")

}

// 实现拦截器  防止 gRPC 服务在处理请求时发生 panic  从而导致程序
func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			// 使用 DPanicLevel，不触发真正的 panic，但可以写入 panic.log
			if ce := zap.L().Check(zapcore.DPanicLevel, "gRPC panic recovered"); ce != nil {
				ce.Write(
					zap.Any("error", r),
					zap.String("method", info.FullMethod),
					zap.ByteString("stack", debug.Stack()),
				)
			}

			// 返回 gRPC 错误响应，不崩溃
			err = status.Error(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}

// 安全执行 goroutine，防止 panic 崩溃
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				zap.S().Errorf("goroutine panic: %v\n堆栈信息:\n%s", r, debug.Stack())
				// 可选报警扩展
			}
		}()
		fn()
	}()
}

// recoverGoRoutine 用于捕获 goroutine 中的 panic 并记录日志
func recoverGoRoutine(name string) {
	if r := recover(); r != nil {
		if ce := zap.L().Check(zapcore.DPanicLevel, fmt.Sprintf("goroutine [%s] panic recovered", name)); ce != nil {
			ce.Write(
				zap.Any("error", r),
				zap.String("goroutine", name),
				zap.ByteString("stack", debug.Stack()),
			)
		}
	}
}
func recoverMainPanic() {
	if r := recover(); r != nil {
		if ce := zap.L().Check(zapcore.DPanicLevel, "main panic recovered"); ce != nil {
			ce.Write(
				zap.Any("error", r),
				zap.ByteString("stack", debug.Stack()),
			)
		}
	}
}
