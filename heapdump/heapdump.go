package heapdump

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"time"

	"go.uber.org/zap"
)

type HeapDumpConfig struct {
	Dir         string        // dump 目录
	Interval    time.Duration // 生成间隔
	MaxFiles    int           // 最大保留文件数
	Enable      bool
	ThresholdMB uint64 // 溢出内存阈值
}

func StartHeapDump(cfg HeapDumpConfig) {
	if !cfg.Enable {
		return
	}

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := dumpOnce(cfg); err != nil {
				zap.S().Error("[HEAP-DUMP] 失败", zap.Error(err))
			}
		}
	}()
}

func dumpOnce(cfg HeapDumpConfig) error {
	// 1. 确保目录存在
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return err
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	zap.S().Debugf("[MEMORY] Alloc = %v MiB | HeapAlloc = %v MiB | Sys = %v MiB | NumGC = %v",
		m.Alloc/1024/1024, m.HeapAlloc/1024/1024, m.Sys/1024/1024, m.NumGC)
	if m.Alloc/1024/1024 < cfg.ThresholdMB {
		return nil
	}
	// 2. 文件名带时间戳
	filename := fmt.Sprintf(
		"heap_%s.prof",
		time.Now().Format("20060102_150405"),
	)
	path := filepath.Join(cfg.Dir, filename)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// 3. 写 heap profile
	if err := pprof.WriteHeapProfile(f); err != nil {
		return err
	}

	// 4. 强制刷盘（关键！）
	if err := f.Sync(); err != nil {
		return err
	}

	zap.S().Warnf("[HEAP-DUMP] 已生成: %s", path)

	// 5. 清理旧文件
	cleanupOldDumps(cfg.Dir, cfg.MaxFiles)

	return nil
}

func cleanupOldDumps(dir string, max int) {
	files, err := os.ReadDir(dir)
	if err != nil || len(files) <= max {
		return
	}

	type fileInfo struct {
		name string
		time time.Time
	}

	var dumps []fileInfo
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		dumps = append(dumps, fileInfo{
			name: f.Name(),
			time: info.ModTime(),
		})
	}

	// 按时间排序
	sort.Slice(dumps, func(i, j int) bool {
		return dumps[i].time.Before(dumps[j].time)
	})

	// 删除多余的
	for i := 0; i < len(dumps)-max; i++ {
		_ = os.Remove(filepath.Join(dir, dumps[i].name))
		zap.S().Warnf("[HEAP-DUMP] 删除旧 dump: %s", dumps[i].name)
	}
}
