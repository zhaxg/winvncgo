package tasks

import (
	_ "embed"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

//go:embed icons/refresh.svg
var refreshSVG []byte

//go:embed icons/copy.svg
var copySVG []byte

var (
	iconRefresh = ui.MustParseSVG(refreshSVG)
	iconCopy    = ui.MustParseSVG(copySVG)
)

// 颜色取自原前端 style.css。
var (
	colorAccent = ui.Hex("#0078D4")
	colorBg    = ui.Hex("#f3f3f3")
	colorLabel   = ui.Hex("#666")
	colorStrong  = ui.Hex("#222")
	colorMuted   = ui.Hex("#999")
	colorGreen   = ui.Hex("#16A34A")
	colorRed     = ui.Hex("#DC2626")
	colorSep     = ui.Hex("#eee")
	colorInput   = ui.Hex("#ddd")
	colorCard    = ui.RGBA(255, 255, 255, 1)
)

// uiState 是界面的全部状态，视图每次按它重建一帧。
type uiState struct {
	app *App
	win *mygo.Window

	ip    string // 设备 IP
	id    string // 本机 ID
	vnc   string // VNC 服务状态
	redis string // 控制中心状态

	remoteID string // 对方 ID
	remoteIP string // 对方 IP（可选）

	showVNC   bool // 状态区是否显示「VNC 服务」（托盘菜单复选项）
	showRedis bool // 状态区是否显示「控制中心」

	toastMsg string    // 顶部提示条内容
	toastAt  time.Time // 提示条出现时刻
}

// newUI 创建界面状态，用后端的初始状态填充。
func newUI(app *App) *uiState {
	u := &uiState{app: app, showVNC: true, showRedis: true}
	st := app.GetStatus()
	if s, ok := st["localIP"].(string); ok {
		u.ip = s
	}
	if s, ok := st["currentId"].(string); ok {
		u.id = s
	}
	if s, ok := st["vnc"].(string); ok {
		u.vnc = s
	}
	if s, ok := st["redis"].(string); ok {
		u.redis = s
	}
	return u
}

// onStatus 是后端状态循环的回调（在 goroutine 上触发），
// 用 Window.Update 切回主线程刷新界面状态。
func (u *uiState) onStatus(s StatusEvent) {
	u.win.Update(func() {
		u.ip, u.id = s.IP, s.ID
		u.vnc, u.redis = s.VNC, s.Redis
	})
}

// view 是窗口视图：由状态构建界面，主线程上每帧调用。
func (u *uiState) view(c *ui.Context) {
	// 固定浅色主题，强调色与原前端一致（Windows 蓝），
	// 字体沿用原前端的 Maple Mono，缺字回退系统字体（中文走回退）
	t := *ui.LightTheme()
	t.Accent = colorAccent
	t.AccentHover = ui.Hex("#106EBE")
	t.AccentPressed = ui.Hex("#005A9E")
	t.Font = "Maple Mono, Segoe UI, Microsoft YaHei"
	c.SetTheme(&t)

	u.toastView(c)

	ui.Column(c).Fill().Background(colorBg).Children(func() {
		// 内容区：两张卡片可滚动
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).FillWidth().Padding(12, 16).Gap(12).Children(func() {
				u.deviceCard(c)
				u.connectCard(c)
			})
		})

		// 页脚
		ui.Text(c, "红河谷 | 红谷软件 版权所有 2026.9").
			FontSize(10).TextColor(colorMuted).TextAlign(ui.Center).
			FillWidth().Padding(10, 0).Opacity(0.5).SingleLine()
	})
}

// card 是统一的白色圆角卡片。
func card(c *ui.Context, title string, body func()) {
	el := ui.Column(c).FillWidth().Radius(10).Padding(20).Background(colorCard)
	el.Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.06))
	el.Children(func() {
		ui.Text(c, title).FontSize(14).Bold().TextColor(colorAccent).Margin(0, 0, 16, 0)
		body()
	})
}

// deviceCard 对应原前端「当前设备」卡片。
func (u *uiState) deviceCard(c *ui.Context) {
	card(c, "当前设备", func() {
		// 设备 IP
		ui.Row(c).AlignItems(ui.Center).Margin(0, 0, 12, 0).Children(func() {
			ui.Text(c, "设备 IP").Width(80).FontSize(13).FontWeight(600).
				TextColor(colorLabel).SingleLine()
			ui.Text(c, u.ip).FontSize(15).Bold().TextColor(colorStrong).
				Grow(1).SingleLine()
		})

		// 设备 ID + 刷新 / 复制
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "设备 ID").Width(80).FontSize(13).FontWeight(600).
				TextColor(colorLabel).SingleLine()
			ui.Text(c, u.id).Font("Maple Mono").FontSize(24).FontWeight(800).
				TextColor(colorAccent).LetterSpacing(2).SingleLine()
			ui.Row(c).Gap(4).Margin(0, 0, 0, 16).Children(func() {
				iconButton(c, iconRefresh, "刷新ID(半小时自动轮换)", u.refreshID)
				iconButton(c, iconCopy, "复制 ID", u.copyID)
			})
		})

		// 分隔线
		ui.Box(c).FillWidth().Height(1).Background(colorSep).Margin(12, 0)

		// 服务状态（显示项由托盘菜单的复选项控制）
		if u.showVNC || u.showRedis {
			ui.Row(c).Gap(24).Children(func() {
				if u.showVNC {
					u.statusItem(c, "VNC 服务", u.vnc, u.vnc == "运行中")
				}
				if u.showRedis {
					u.statusItem(c, "控制中心", u.redis, u.redis == "已连接")
				}
			})
		}
	})
}

// statusItem 是「标签 + 圆点 + 状态文字」的一组状态。
func (u *uiState) statusItem(c *ui.Context, label, value string, ok bool) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(0, 0, 0, 12).
		FontSize(12).Children(func() {
		ui.Text(c, label).TextColor(colorMuted).SingleLine()
		dot := ui.Box(c).Size(8, 8).Radius(4)
		if ok {
			dot.Background(colorGreen)
		} else {
			dot.Background(colorRed)
		}
		col := colorGreen
		if !ok {
			col = colorRed
		}
		ui.Text(c, value).TextColor(col).FontWeight(600).SingleLine()
	})
}

// iconButton 是无边框图标按钮，带悬停底色与提示气泡。
func iconButton(c *ui.Context, icon *ui.SVG, tip string, onClick func()) {
	b := ui.ButtonBase(c)
	b.Size(32, 32).Radius(6).Tooltip(tip).Cursor(ui.CursorPointer)
	switch {
	case b.Pressed():
		b.Background(ui.Hex("#e0e0e0"))
	case b.Hovered():
		b.Background(ui.Hex("#f0f0f0"))
	}
	b.Children(func() {
		ui.Image(c, icon).Size(16, 16)
	})
	if b.Clicked() {
		onClick()
	}
}

// connectCard 对应原前端「控制他人设备」卡片。
func (u *uiState) connectCard(c *ui.Context) {
	card(c, "控制他人设备", func() {
		u.inputGroup(c, "设备 ID", "输入对方 6 位 ID", &u.remoteID, 6)
		u.inputGroup(c, "设备 IP（可选）", "输入对方地址IP[:port]", &u.remoteIP, 0)

		btn := ui.PrimaryButton(c, "连 接").FillWidth().
			Padding(12).Radius(8).FontSize(14).FontWeight(600)
		if btn.Clicked() {
			u.connect()
		}
	})
}

// inputGroup 是「标签 + 输入框」一组；limit>0 时限定字符数。
func (u *uiState) inputGroup(c *ui.Context, label, placeholder string, value *string, limit int) {
	ui.Column(c).Margin(0, 0, 14, 0).Children(func() {
		ui.Text(c, label).FontSize(13).FontWeight(600).TextColor(colorLabel).
			Margin(0, 0, 6, 0).SingleLine()
		in := ui.TextInput(c, value).Placeholder(placeholder).FillWidth().
			Padding(10, 14).Radius(8).Border(1, colorInput).
			Background(colorCard).FontSize(14)
		if in.Focused() {
			in.BorderColor(colorAccent)
		}
		if limit > 0 && in.Changed() {
			if r := []rune(*value); len(r) > limit {
				*value = string(r[:limit])
			}
		}
	})
}

// ── 操作：全部走后端 App 方法 ──

// refreshID 刷新设备 ID（后端 RefreshID）。
func (u *uiState) refreshID() {
	res := u.app.RefreshID()
	if ok, _ := res["ok"].(bool); ok {
		if id, ok := res["id"].(string); ok && id != "" {
			u.id = id
		}
		u.showToast("ID 已刷新: " + u.id)
		return
	}
	u.showToast(resultError(res, "刷新失败"))
}

// copyID 把当前 ID 写入剪贴板。
func (u *uiState) copyID() {
	id := u.app.CopyID()
	mygo.Clipboard.WriteText(id)
	u.showToast("ID 已复制: " + id)
}

// connect 连接远程设备（后端 Connect）。
func (u *uiState) connect() {
	remoteID := strings.TrimSpace(u.remoteID)
	if remoteID == "" {
		u.showToast("请输入远程 ID")
		return
	}
	res := u.app.Connect(remoteID, strings.TrimSpace(u.remoteIP))
	if ok, _ := res["ok"].(bool); ok {
		u.showToast("已发起连接")
		return
	}
	u.showToast(resultError(res, "连接失败"))
}

// resultError 取结果 map 里的 error 字段，缺省用 fallback。
func resultError(res map[string]interface{}, fallback string) string {
	if s, ok := res["error"].(string); ok && s != "" {
		return s
	}
	return fallback
}

// showToast 在窗口顶部弹出绿色提示条，2 秒后淡出。
func (u *uiState) showToast(msg string) {
	u.toastMsg, u.toastAt = msg, time.Now()
}

// toastView 绘制提示条并调度淡出后的重绘。
func (u *uiState) toastView(c *ui.Context) {
	if u.toastMsg == "" {
		return
	}
	const life = 2 * time.Second
	const fade = 300 * time.Millisecond
	age := time.Since(u.toastAt)
	if age >= life {
		u.toastMsg = ""
		return
	}
	opacity := float32(1)
	switch left := life - age; {
	case age < fade:
		opacity = float32(age) / float32(fade)
		c.AnimationFrame()
	case left < fade:
		opacity = float32(left) / float32(fade)
		c.AnimationFrame()
	default:
		c.After(left - fade)
	}
	ui.Overlay(c, func() {
		stack := ui.Column(c).Absolute().
			Left(0).Right(0).Top(50).
			AlignItems(ui.Center).PassThrough()
		stack.Children(func() {
			box := ui.Row(c).Padding(8, 20).Radius(6).
				Background(colorGreen).
				Shadow(0, 2, 12, 0, ui.RGBA(0, 0, 0, 0.15)).
				Role(ui.RoleStatus)
			box.Opacity(opacity)
			box.Children(func() {
				ui.Text(c, u.toastMsg).FontSize(13).FontWeight(600).
					TextColor(ui.RGB(255, 255, 255)).SingleLine()
			})
		})
	})
}
