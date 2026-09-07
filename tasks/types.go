package tasks

// StatusEvent 状态事件，用于前端显示
type StatusEvent struct {
	IP    string `json:"ip"`
	ID    string `json:"id"`
	VNC   string `json:"vnc"`
	Redis string `json:"redis"`
	Conn  string `json:"connection"`
}
