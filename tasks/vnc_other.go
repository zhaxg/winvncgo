//go:build !windows

package tasks

type VNCService struct{}

func newVNCService(baseDir string) *VNCService { return nil }

func (s *VNCService) EnsureService() error              { return nil }
func (s *VNCService) SetPassword(id string) error        { return nil }
func (s *VNCService) Stop()                             {}
func (s *VNCService) IsRunning() bool                   { return false }
func (s *VNCService) RefreshConnectionState()           {}
func (s *VNCService) HasActiveConnection() bool         { return false }
func (s *VNCService) LaunchViewer(ip string, port int, password string) error {
	return nil
}

func hasEstablishedConnections(port int) bool { return false }

// replaceSection 非 Windows 平台空实现
func replaceSection(content, section, newSection string) string { return content }

// splitLines 非 Windows 平台空实现
func splitLines(s string) []string { return nil }
