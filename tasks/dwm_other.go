//go:build !windows

package tasks

// applyBackdrop 非 Windows 平台的空实现
func applyBackdrop(hwnd uintptr, backdropType int) {}

// findWindowByTitle 非 Windows 平台的空实现
func findWindowByTitle(title string) uintptr { return 0 }
