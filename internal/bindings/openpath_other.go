//go:build !windows

// 非 Windows 平台的文件管理器回退实现：直接调用系统默认命令，
// 无窗口复用能力（与原行为一致）。
package bindings

import (
	"os/exec"
	"runtime"
)

// openFileInFileManager 在文件管理器中打开文件所在目录（不做选中高亮）。
func openFileInFileManager(path string) error {
	return openInFileManager(path)
}

// openInFileManager 打开目录。
func openInFileManager(dir string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}
