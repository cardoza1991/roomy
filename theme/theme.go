// theme.go

package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Define and Export CustomTheme struct
type CustomTheme struct{}

var _ fyne.Theme = (*CustomTheme)(nil) // Interface assertion

// Exported Colors and Icons
var (
	ButtonColor       = color.NRGBA{R: 108, G: 122, B: 137, A: 255} // Button background color
	TextColor         = color.NRGBA{R: 33, G: 37, B: 41, A: 255}    // Text color
	DisabledTextColor = color.NRGBA{R: 173, G: 181, B: 189, A: 255} // Disabled text color
	Padding           = float32(10)                                 // Padding size for UI elements

	// Icons
	ContentCopyIcon    fyne.Resource = theme.ContentCopyIcon()
	SearchIcon         fyne.Resource = theme.SearchIcon()
	SettingsIcon       fyne.Resource = theme.SettingsIcon()
	LogoutIcon         fyne.Resource = theme.LogoutIcon()
	LoginIcon          fyne.Resource = theme.LoginIcon()
	DocumentCreateIcon fyne.Resource = theme.DocumentCreateIcon()
)

// Use LightTheme as the default to avoid calling fyne.CurrentApp()
var defaultTheme = theme.LightTheme()

func (c *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameButton:
		return ButtonColor
	case theme.ColorNameForeground:
		return TextColor
	case theme.ColorNameDisabled:
		return DisabledTextColor
	default:
		return defaultTheme.Color(name, variant)
	}
}

func (c *CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return defaultTheme.Font(style)
}

func (c *CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return defaultTheme.Icon(name)
}

func (c *CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return Padding
	default:
		return defaultTheme.Size(name)
	}
}
