package tasks

import (
	_ "embed"

	"github.com/egoist/mygo"
)

//go:embed icons/appicon.png
var appIconPNG []byte

//go:embed icons/tray.png
var trayIconPNG []byte

// setupTray 创建托盘图标及其右键菜单。
// 菜单里「连接」「状态显示」是带子菜单的多级菜单。
func (u *uiState) setupTray(win *mygo.Window) {
	appLog("tray: 构建菜单")
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "打开主窗口", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
			win.Show()
			win.Focus()
		}},
		mygo.Separator(),
		{Label: "刷新设备 ID", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
			win.Update(func() { u.refreshID() })
		}},
		{Label: "复制设备 ID", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
			win.Update(func() { u.copyID() })
		}},
		mygo.Separator(),
		// 二级：连接
		{Label: "连接", Submenu: []*mygo.MenuItem{
			{Label: "连接上次设备", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				win.Update(func() {
					if u.remoteID == "" {
						u.showToast("没有历史连接记录")
						return
					}
					u.connect()
				})
			}},
			// 三级：清空输入
			{Label: "清空连接输入", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				win.Update(func() {
					u.remoteID, u.remoteIP = "", ""
					u.showToast("已清空输入")
				})
			}},
			mygo.Separator(),
			{Label: "断开当前连接", Disabled: true},
		}},
		// 二级：状态显示（复选框，控制窗口状态区）
		{Label: "状态显示", Submenu: []*mygo.MenuItem{
			{Label: "VNC 服务", Type: mygo.MenuItemCheckbox, Checked: true,
				Click: func(it *mygo.MenuItem, _ *mygo.Window) {
					// 勾选状态在 Click 之前已切换，直接同步到视图状态
					v := it.IsChecked()
					win.Update(func() { u.showVNC = v })
				}},
			{Label: "控制中心", Type: mygo.MenuItemCheckbox, Checked: true,
				Click: func(it *mygo.MenuItem, _ *mygo.Window) {
					v := it.IsChecked()
					win.Update(func() { u.showRedis = v })
				}},
		}},
		mygo.Separator(),
		{Label: "退出", Role: mygo.RoleQuit},
	})

	tray, err := mygo.NewTray(mygo.TrayOptions{
		Icon:    trayIconPNG,
		ToolTip: "远程协助",
		Menu:    menu,
	})
	if err != nil {
		appLogf("tray: NewTray 失败: %v", err)
		return
	}
	appLogf("tray: ok (%p)", tray)
}
