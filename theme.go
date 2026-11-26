package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Apple's light appearance: a pale desktop tinted lavender and blue, with
// frosted glass chrome over it. glass.Glass draws the frost in view.go.
var (
	appleCanvasTop   = ui.RGB(241, 243, 249)
	appleCanvasEdge  = ui.RGB(214, 221, 234)
	appleChrome      = ui.RGBA(255, 255, 255, 0.62)
	appleChromeSolid = ui.RGB(249, 250, 252)
	appleElevated    = ui.RGBA(255, 255, 255, 0.94)
	appleHoverFill   = ui.RGBA(0, 0, 0, 0.055)
	appleLine        = ui.RGBA(0, 0, 0, 0.14)
	appLineStrong    = ui.RGBA(0, 0, 0, 0.22)
	appleText        = ui.RGB(29, 29, 31)
	appleTextMuted   = ui.RGB(81, 81, 84)
	appleTextDim     = ui.RGB(134, 134, 139)
	appleTextFaint   = ui.RGB(174, 174, 178)
	appleBlue        = ui.RGB(0, 122, 255)
	appleBlueHover   = ui.RGB(0, 113, 227)
	appleBlueActive  = ui.RGB(0, 96, 213)
	appleBlueFill    = ui.RGBA(0, 122, 255, 0.20)
	appleBlueFillDim = ui.RGBA(0, 122, 255, 0.09)
	appleGreen       = ui.RGB(52, 199, 89)
	appleAmber       = ui.RGB(255, 179, 10)
	appleRed         = ui.RGB(255, 59, 48)
)

var appTheme = newAppTheme()

func newAppTheme() *theme.Config {
	light := *theme.Default().Tokens(false)

	light.BgLayout = ui.RGB(247, 248, 251)
	light.BgBase = appleCanvasEdge
	light.BgContainer = appleChrome
	light.BgContainerSecondary = ui.RGBA(255, 255, 255, 0.46)
	light.BgElevated = appleElevated
	light.BgSolid = ui.RGBA(255, 255, 255, 0.92)
	light.BgSolidHover = ui.RGBA(255, 255, 255, 0.98)
	light.BgSolidActive = ui.RGBA(238, 240, 245, 0.96)
	light.BgContainerDisabled = ui.RGBA(245, 246, 249, 0.62)
	light.BgMask = ui.RGBA(18, 22, 30, 0.30)
	light.BgBlur = ui.RGBA(250, 251, 253, 0.70)

	light.Border = appleLine
	light.BorderSecondary = ui.RGBA(0, 0, 0, 0.075)
	light.BorderDisabled = ui.RGBA(0, 0, 0, 0.05)
	light.BorderBg = ui.RGBA(0, 0, 0, 0.10)

	light.Fill = ui.RGBA(0, 0, 0, 0.12)
	light.FillSecondary = ui.RGBA(0, 0, 0, 0.075)
	light.FillTertiary = ui.RGBA(0, 0, 0, 0.045)
	light.FillQuaternary = ui.RGBA(0, 0, 0, 0.025)
	light.FillAlter = ui.RGBA(0, 0, 0, 0.035)
	light.FillContent = ui.RGBA(255, 255, 255, 0.55)
	light.FillContentHover = ui.RGBA(255, 255, 255, 0.78)

	light.Text = appleText
	light.TextBase = appleText
	light.TextHeading = ui.RGB(0, 0, 0)
	light.TextLabel = appleText
	light.TextSecondary = appleTextMuted
	light.TextTertiary = appleTextDim
	light.TextDescription = appleTextMuted
	light.TextQuaternary = appleTextDim
	light.TextPlaceholder = ui.RGB(158, 158, 163)
	light.TextDisabled = ui.RGB(188, 188, 192)
	light.TextLightSolid = ui.RGB(255, 255, 255)

	light.Info = appleBlue
	light.InfoHover = appleBlueHover
	light.InfoActive = appleBlueActive
	light.InfoBg = appleBlueFillDim
	light.InfoBgHover = appleBlueFill
	light.InfoBorder = ui.RGBA(0, 122, 255, 0.42)
	light.InfoBorderHover = appleBlueHover
	light.InfoFill = appleBlueFill
	light.InfoFillSecondary = appleBlueFillDim
	light.InfoFillTertiary = ui.RGBA(0, 122, 255, 0.06)
	light.InfoFillQuaternary = ui.RGBA(0, 122, 255, 0.03)
	light.InfoText = appleBlue
	light.InfoTextHover = appleBlueHover
	light.InfoTextActive = appleBlueActive
	light.Link = appleBlue
	light.LinkHover = appleBlueHover
	light.LinkActive = appleBlueActive
	light.Blue = appleBlue
	light.BlueActive = appleBlueActive
	light.BlueFillTertiary = ui.RGBA(0, 122, 255, 0.06)
	light.BlueFillQuaternary = ui.RGBA(0, 122, 255, 0.03)

	light.Primary = appleBlue
	light.PrimaryHover = appleBlueHover
	light.PrimaryActive = appleBlueActive
	light.PrimaryBg = appleBlueFill
	light.PrimaryBgHover = ui.RGBA(0, 122, 255, 0.26)
	light.PrimaryBorder = ui.RGBA(0, 122, 255, 0.50)
	light.PrimaryBorderHover = appleBlueHover
	light.PrimaryFill = appleBlueFill
	light.PrimaryFillSecondary = appleBlueFillDim
	light.PrimaryFillTertiary = ui.RGBA(0, 122, 255, 0.06)
	light.PrimaryFillQuaternary = ui.RGBA(0, 122, 255, 0.03)
	light.PrimaryForeground = ui.RGB(255, 255, 255)
	light.PrimaryText = appleBlue
	light.PrimaryTextHover = appleBlueHover
	light.PrimaryTextActive = appleBlueActive

	light.Success = appleGreen
	light.SuccessHover = ui.RGB(40, 180, 72)
	light.SuccessActive = ui.RGB(32, 150, 60)
	light.SuccessText = ui.RGB(36, 160, 62)
	light.Warning = appleAmber
	light.WarningHover = ui.RGB(240, 160, 0)
	light.WarningActive = ui.RGB(210, 140, 0)
	light.WarningText = ui.RGB(190, 120, 0)
	light.Error = appleRed
	light.ErrorHover = ui.RGB(230, 45, 35)
	light.ErrorActive = ui.RGB(200, 32, 24)
	light.ErrorText = ui.RGB(235, 45, 35)
	light.ErrorBg = ui.RGBA(255, 59, 48, 0.09)
	light.ErrorBgHover = ui.RGBA(255, 59, 48, 0.16)

	light.Split = ui.RGBA(0, 0, 0, 0.08)
	light.Shadow = ui.RGBA(28, 36, 52, 0.22)
	light.Highlight = ui.RGBA(255, 255, 255, 0.58)

	light.BorderRadius = 8
	light.BorderRadiusLG = 14
	light.BorderRadiusSM = 6
	light.BorderRadiusXS = 4
	light.ControlHeight = 32
	light.ControlHeightLG = 40
	light.ControlHeightSM = 26
	light.ControlHeightXS = 22
	light.FontSize = 13
	light.FontSizeHeading1 = 30
	light.FontSizeHeading2 = 24
	light.FontSizeHeading3 = 20
	light.FontSizeHeading4 = 16
	light.FontSizeHeading5 = 14

	dark := light
	dark.Dark = true
	cfg := theme.NewConfig(light, dark)
	for _, t := range []*ui.Theme{cfg.Theme(false), cfg.Theme(true)} {
		t.Dark = false
		t.Background = appleCanvasEdge
		t.Surface = ui.RGBA(255, 255, 255, 0.64)
		t.SurfaceHover = ui.RGBA(255, 255, 255, 0.88)
		t.SurfacePressed = ui.RGBA(235, 238, 244, 0.9)
		t.Border = appleLine
		t.Text = appleText
		t.TextMuted = appleTextDim
		t.Accent = appleBlue
		t.AccentHover = appleBlueHover
		t.AccentPressed = appleBlueActive
		t.AccentText = ui.RGB(255, 255, 255)
		t.InactiveAccent = ui.RGBA(196, 199, 207, 0.92)
		t.Danger = appleRed
		t.Warning = appleAmber
		t.Success = appleGreen
		t.Selection = ui.RGBA(0, 122, 255, 0.24)
		t.Focus = ui.RGBA(0, 122, 255, 0.80)
		t.FocusRingWidth = 2
		t.FocusRingOffset = 2
		t.Inverse = ui.RGBA(32, 36, 44, 0.94)
		t.InverseText = ui.RGB(245, 246, 248)
		t.Scrollbar = ui.RGBA(0, 0, 0, 0.24)
		t.ScrollbarWidth = 9
		t.Radius = 8
		t.Spacing = 4
		t.FontSize = 13
		t.Font = "SF Pro Text, Segoe UI Variable, Segoe UI, sans-serif"
		t.TooltipDelay = 0
	}
	return cfg
}

// applyAppleUI pins the app to the Apple light appearance. It is the
// product's look, not an optional desktop theme: the frosted chrome and the
// pastel desktop are built on top of it.
func applyAppleUI(c *ui.Context) {
	c.SetTheme(appTheme.Theme(false))
}

func vline(c *ui.Context, tk *theme.Tokens) {
	ui.Box(c).Height(1).FillWidth().Background(tk.BorderSecondary)
}

func vbar(c *ui.Context, tk *theme.Tokens) {
	ui.Box(c).Width(1).FillHeight().Background(tk.BorderSecondary)
}

func eyebrow(c *ui.Context, tk *theme.Tokens, text string) {
	ui.Text(c, text).FontSize(10).FontWeight(600).LetterSpacing(0.8).
		TextColor(tk.TextQuaternary).PaddingX(6).PaddingY(4)
}
