// Package i18n provides internationalization support for my-pnpm-installer.
//
// This package handles:
//   - Loading translations for UI text
//   - Language detection from environment
//   - Fallback to default language (Chinese)
//   - Dynamic translation loading and validation
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Language represents a supported language
type Language string

const (
	// Chinese is the default language
	Chinese Language = "zh"
	// English is the alternative language
	English Language = "en"
)

// Translations holds all UI text translations
type Translations struct {
	// List view
	ListTitle           string
	ListHelp            string
	SearchPlaceholder   string
	LoadingPackageInfo  string
	NoPackageSelected   string
	
	// Package details
	PackageLabel        string
	LatestVersionLabel  string
	InstalledLabel      string
	StatusLabel         string
	
	// Status values
	StatusInstalled     string
	StatusNotInstalled  string
	StatusUpdateAvail   string
	StatusUnknown       string
	
	// Install view
	InstallingTitle     string
	InstallLogs         string
	PressQToReturn      string
	
	// Help text
	HelpNavigation      string
	HelpInstall         string
	HelpRefresh         string
	HelpQuit            string
}

var (
	// Current language
	currentLang Language = Chinese

	// Translation maps
	translations = map[Language]Translations{
		Chinese: {
			ListTitle:          "包管理器",
			ListHelp:           "↑/↓ 或 j/k: 导航 | J/K: 滚动详情 | Enter: 安装 | r: 刷新 | /: 过滤 | q: 退出",
			SearchPlaceholder:  "搜索包...",
			LoadingPackageInfo: "⏳ 正在加载包信息...",
			NoPackageSelected:  "请从左侧列表选择一个包",

			PackageLabel:       "包名",
			LatestVersionLabel: "最新版本",
			InstalledLabel:     "已安装版本",
			StatusLabel:        "状态",

			StatusInstalled:    "✓ 已安装（最新版本）",
			StatusNotInstalled: "✗ 未安装",
			StatusUpdateAvail:  "↑ 有更新可用",
			StatusUnknown:      "? 未知",

			InstallingTitle:    "正在安装",
			InstallLogs:        "安装日志",
			PressQToReturn:     "按 q 返回列表",

			HelpNavigation:     "↑/↓ 或 j/k 导航",
			HelpInstall:        "Enter 安装/更新",
			HelpRefresh:        "r 刷新",
			HelpQuit:           "q 退出",
		},
		English: {
			ListTitle:          "Package Manager",
			ListHelp:           "↑/↓ or j/k: navigate | J/K: scroll details | Enter: install | r: refresh | /: filter | q: quit",
			SearchPlaceholder:  "Search packages...",
			LoadingPackageInfo: "⏳ Loading package information...",
			NoPackageSelected:  "Select a package from the list",

			PackageLabel:       "Package",
			LatestVersionLabel: "Latest Version",
			InstalledLabel:     "Installed Version",
			StatusLabel:        "Status",

			StatusInstalled:    "✓ Installed (up to date)",
			StatusNotInstalled: "✗ Not installed",
			StatusUpdateAvail:  "↑ Update available",
			StatusUnknown:      "? Unknown",

			InstallingTitle:    "Installing",
			InstallLogs:        "Installation Logs",
			PressQToReturn:     "Press q to return to list",

			HelpNavigation:     "↑/↓ or j/k to navigate",
			HelpInstall:        "Enter to install/update",
			HelpRefresh:        "r to refresh",
			HelpQuit:           "q to quit",
		},
	}

	// Mutex for thread-safe translation operations
	translationsMu sync.RWMutex

	// defaultTranslations is used as a reference for validation
	defaultTranslations Translations
)

func init() {
	// Store Chinese translations as default for validation
	defaultTranslations = translations[Chinese]
}

// Init initializes the i18n system by detecting the user's language
func Init() {
	currentLang = DetectLanguage()
}

// DetectLanguage detects the user's preferred language from environment variables
func DetectLanguage() Language {
	// Check LANG environment variable (common on Unix systems)
	lang := os.Getenv("LANG")
	if lang == "" {
		// Check LC_ALL as fallback
		lang = os.Getenv("LC_ALL")
	}
	
	// Parse language code (e.g., "zh_CN.UTF-8" -> "zh")
	lang = strings.ToLower(lang)
	if strings.HasPrefix(lang, "zh") {
		return Chinese
	}
	if strings.HasPrefix(lang, "en") {
		return English
	}
	
	// Default to Chinese
	return Chinese
}

// SetLanguage sets the current language
func SetLanguage(lang Language) {
	currentLang = lang
}

// GetLanguage returns the current language
func GetLanguage() Language {
	return currentLang
}

// T returns the translations for the current language
func T() Translations {
	translationsMu.RLock()
	defer translationsMu.RUnlock()
	return translations[currentLang]
}

// LoadTranslations loads translations for a new language dynamically.
// This allows adding new languages at runtime without recompiling.
//
// Example:
//
//	i18n.LoadTranslations(i18n.Language("ja"), Translations{
//	    ListTitle: "パッケージマネージャー",
//	    // ... other translations
//	})
func LoadTranslations(lang Language, data Translations) error {
	if err := validateTranslations(data); err != nil {
		return fmt.Errorf("invalid translations for language %s: %w", lang, err)
	}

	translationsMu.Lock()
	defer translationsMu.Unlock()

	translations[lang] = data
	return nil
}

// validateTranslations checks if all required translation fields are populated
func validateTranslations(t Translations) error {
	// Use reflection-free approach by checking key fields
	if t.ListTitle == "" {
		return fmt.Errorf("ListTitle is required")
	}
	if t.PackageLabel == "" {
		return fmt.Errorf("PackageLabel is required")
	}
	if t.StatusInstalled == "" {
		return fmt.Errorf("StatusInstalled is required")
	}
	return nil
}

// ValidateTranslations validates all loaded translations for completeness.
// Returns a list of validation errors for each language.
func ValidateTranslations() map[Language][]string {
	translationsMu.RLock()
	defer translationsMu.RUnlock()

	errors := make(map[Language][]string)

	for lang, t := range translations {
		var langErrors []string

		// Check each field
		if t.ListTitle == "" {
			langErrors = append(langErrors, "ListTitle is empty")
		}
		if t.ListHelp == "" {
			langErrors = append(langErrors, "ListHelp is empty")
		}
		if t.PackageLabel == "" {
			langErrors = append(langErrors, "PackageLabel is empty")
		}
		if t.LatestVersionLabel == "" {
			langErrors = append(langErrors, "LatestVersionLabel is empty")
		}
		if t.InstalledLabel == "" {
			langErrors = append(langErrors, "InstalledLabel is empty")
		}
		if t.StatusLabel == "" {
			langErrors = append(langErrors, "StatusLabel is empty")
		}
		if t.StatusInstalled == "" {
			langErrors = append(langErrors, "StatusInstalled is empty")
		}
		if t.StatusNotInstalled == "" {
			langErrors = append(langErrors, "StatusNotInstalled is empty")
		}
		if t.StatusUpdateAvail == "" {
			langErrors = append(langErrors, "StatusUpdateAvail is empty")
		}
		if t.InstallingTitle == "" {
			langErrors = append(langErrors, "InstallingTitle is empty")
		}

		if len(langErrors) > 0 {
			errors[lang] = langErrors
		}
	}

	return errors
}

// GetSupportedLanguages returns a list of all supported language codes
func GetSupportedLanguages() []Language {
	translationsMu.RLock()
	defer translationsMu.RUnlock()

	languages := make([]Language, 0, len(translations))
	for lang := range translations {
		languages = append(languages, lang)
	}
	return languages
}

// DetectLanguageFromLocale detects language from a locale string (e.g., "zh_CN.UTF-8", "en_US")
func DetectLanguageFromLocale(locale string) Language {
	locale = strings.ToLower(locale)

	// Handle common locale formats
	localeMappings := map[string]Language{
		"zh": Chinese, "zh_cn": Chinese, "zh_tw": Chinese, "zh_hk": Chinese,
		"en": English, "en_us": English, "en_gb": English, "en_au": English,
	}

	// Try exact match first
	if lang, ok := localeMappings[locale]; ok {
		return lang
	}

	// Try prefix match (e.g., "zh_CN" -> "zh")
	for prefix, lang := range localeMappings {
		if strings.HasPrefix(locale, prefix) {
			return lang
		}
	}

	// Default to Chinese
	return Chinese
}
