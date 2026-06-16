package api

import (
	"fmt"
	"strings"
	"sync"
	"time"

	sapi "github.com/striter-no/softgo/api"
	"github.com/striter-no/stg/graphics"
)

type DebugConsole struct {
	mu sync.Mutex

	screen  *sapi.RenderScreen
	enabled bool

	lines    []string
	maxLines int

	posX, posY int

	counters []string

	lastTickAt time.Time
	fpsEMA     float32
}

func NewDebugConsole(screen *sapi.RenderScreen, maxLines int) *DebugConsole {
	if maxLines <= 0 {
		maxLines = 12
	}
	return &DebugConsole{
		screen:   screen,
		enabled:  false,
		lines:    make([]string, 0, maxLines),
		maxLines: maxLines,
		posX:     0,
		posY:     10,
	}
}

func (c *DebugConsole) Enable()  { c.mu.Lock(); c.enabled = true; c.mu.Unlock() }
func (c *DebugConsole) Disable() { c.mu.Lock(); c.enabled = false; c.mu.Unlock() }
func (c *DebugConsole) Enabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabled
}

func (c *DebugConsole) SetPos(x, y int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.posX = x
	c.posY = y
}

func (c *DebugConsole) SetMaxLines(n int) {
	if n <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxLines = n
	if len(c.lines) > n {
		c.lines = c.lines[len(c.lines)-n:]
	}
	c.lines = c.lines[:len(c.lines):cap(c.lines)]
}

func (c *DebugConsole) Log(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	line := fmt.Sprintf(format, args...)
	if len(c.lines) >= c.maxLines {

		c.lines = c.lines[1:]
	}
	c.lines = append(c.lines, line)
}

func (c *DebugConsole) SetCounters(lines []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters = lines
}

func (c *DebugConsole) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = c.lines[:0]
}

func (c *DebugConsole) Render() {
	c.mu.Lock()
	if !c.enabled {
		c.mu.Unlock()
		return
	}

	counters := make([]string, len(c.counters))
	copy(counters, c.counters)
	lines := make([]string, len(c.lines))
	copy(lines, c.lines)
	x, y := c.posX, c.posY
	c.mu.Unlock()

	pixel := graphics.NewFGPixel(255, 255, 255, "")

	for i, ln := range counters {
		c.screen.Screen.SetText(x, y+i, ln, pixel)
	}

	offset := len(counters)
	for i, ln := range lines {
		c.screen.Screen.SetText(x, y+offset+i, ln, pixel)
	}
}

func (c *DebugConsole) Tick() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if !c.lastTickAt.IsZero() {
		dt := now.Sub(c.lastTickAt).Seconds()
		if dt > 0 {
			fps := 1.0 / dt
			if c.fpsEMA == 0 {
				c.fpsEMA = float32(fps)
			} else {
				c.fpsEMA = c.fpsEMA*0.9 + float32(fps)*0.1
			}
		}
	}
	c.lastTickAt = now
}

func (c *DebugConsole) FPS() float32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fpsEMA
}

var globalConsole *DebugConsole

func SetGlobalDebugConsole(c *DebugConsole) {
	globalConsole = c
}

func DebugLog(format string, args ...any) {
	if globalConsole != nil {
		globalConsole.Log(format, args...)
	}
}

var _ = strings.Repeat
