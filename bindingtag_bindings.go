//go:build bindings

package main

// bindingGeneration 在 Wails 绑定生成模式下为 true。
// wails build 生成绑定时会以 -tags bindings 编译并执行本程序（不启动窗口）。
const bindingGeneration = true
