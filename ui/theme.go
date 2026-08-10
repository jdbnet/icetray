//go:build linux && !headless

package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Brand colors taken from the IceTray icon.
var (
	colorMint      = color.NRGBA{R: 0, G: 229, B: 160, A: 255}
	colorMintHover   = color.NRGBA{R: 51, G: 240, B: 184, A: 255}
	colorBackground  = color.NRGBA{R: 10, G: 10, B: 10, A: 255}
	colorSurface     = color.NRGBA{R: 22, G: 22, B: 22, A: 255}
	colorBorder      = color.NRGBA{R: 42, G: 42, B: 42, A: 255}
	colorText        = color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	colorTextMuted   = color.NRGBA{R: 140, G: 140, B: 140, A: 255}
	colorError       = color.NRGBA{R: 255, G: 82, B: 82, A: 255}
)

type mintTheme struct {
	base fyne.Theme
}

func newMintTheme() fyne.Theme {
	return &mintTheme{base: theme.DefaultTheme()}
}

func (t *mintTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colorBackground
	case theme.ColorNameButton:
		return colorMint
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0, G: 229, B: 160, A: 128}
	case theme.ColorNameForeground:
		return colorText
	case theme.ColorNameInputBackground:
		return colorSurface
	case theme.ColorNameInputBorder:
		return colorBorder
	case theme.ColorNamePlaceHolder:
		return colorTextMuted
	case theme.ColorNamePrimary:
		return colorMint
	case theme.ColorNameHover:
		return colorMintHover
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 180}
	case theme.ColorNameError:
		return colorError
	}

	return t.base.Color(name, theme.VariantDark)
}

func (t *mintTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *mintTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *mintTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.base.Size(name)
}
