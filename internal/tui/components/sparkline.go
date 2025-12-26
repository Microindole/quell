package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Sparkline 是一个纯 UI 组件，只负责渲染数据
type Sparkline struct {
	Style lipgloss.Style
}

func NewSparkline(style lipgloss.Style) *Sparkline {
	return &Sparkline{Style: style}
}

// Render 接收数据并返回渲染后的字符串
func (s *Sparkline) Render(data []float64) string {
	if len(data) == 0 {
		return ""
	}

	m := 0.0
	for _, v := range data {
		if v > m {
			m = v
		}
	}

	// 避免最大值为 0 或极小时导致除零或噪点放大
	if m < 1.0 {
		m = 1.0
	}

	levels := []rune(" ▂▃▄▅▆▇█")
	var sb strings.Builder

	for _, v := range data {
		ratio := v / m
		idx := int(ratio * float64(len(levels)-1))

		if idx < 0 {
			idx = 0
		}
		if idx >= len(levels) {
			idx = len(levels) - 1
		}

		sb.WriteRune(levels[idx])
	}

	return s.Style.Render(sb.String())
}
