//go:build !linux && !darwin

package caddy

import "os"

// fileIdentity 在无 inode 概念的平台（如 Windows）上无法判断文件替换，
// 恒定返回 0，调用方据此按「未知」处理，退化为原有的按偏移量恢复逻辑。
func fileIdentity(_ os.FileInfo) uint64 {
	return 0
}
