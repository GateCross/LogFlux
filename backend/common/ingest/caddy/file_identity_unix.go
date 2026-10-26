//go:build linux || darwin

package caddy

import (
	"os"
	"syscall"
)

// fileIdentity 把文件 inode 标识压成一个 uint64 值，用于判断
// 「同一路径下的文件是否已被替换」。
//
// 取 dev 与 ino 的低 32 位拼接：同一次进程生命周期内足以区分日志滚动
// 产生的新文件，又避开了 32 位平台 int64 转换的溢出问题。
// 取不到时返回 0，调用方按「未知」处理而不是判定为变化。
func fileIdentity(info os.FileInfo) uint64 {
	if info == nil {
		return 0
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	//nolint:unconvert // dev/ino 在部分平台为 int32，统一转 uint64 避免符号扩展
	dev := uint64(stat.Dev)
	//nolint:unconvert
	ino := uint64(stat.Ino)
	return (dev << 32) | (ino & 0xFFFFFFFF)
}
