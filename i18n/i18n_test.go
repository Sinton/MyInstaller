package i18n

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	// Note: We can't easily modify env vars in tests,
	// so we test DetectLanguageFromLocale instead
	tests := []struct {
		name     string
		locale   string
		expected Language
	}{
		{
			name:     "Chinese locale",
			locale:   "zh_CN.UTF-8",
			expected: Chinese,
		},
		{
			name:     "English locale",
			locale:   "en_US.UTF-8",
			expected: English,
		},
		{
			name:     "Default to Chinese",
			locale:   "",
			expected: Chinese,
		},
		{
			name:     "Unknown language defaults to Chinese",
			locale:   "fr_FR.UTF-8",
			expected: Chinese,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectLanguageFromLocale(tt.locale)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDetectLanguageFromLocale(t *testing.T) {
	tests := []struct {
		locale   string
		expected Language
	}{
		// Chinese locales
		{"zh_CN.UTF-8", Chinese},
		{"zh_TW.UTF-8", Chinese},
		{"zh_HK.UTF-8", Chinese},
		{"zh", Chinese},
		{"zh_CN", Chinese},
		
		// English locales
		{"en_US.UTF-8", English},
		{"en_GB.UTF-8", English},
		{"en_AU.UTF-8", English},
		{"en", English},
		{"en_US", English},
		
		// Edge cases
		{"", Chinese}, // Default
		{"invalid", Chinese}, // Default
		{"ja_JP.UTF-8", Chinese}, // Unknown defaults to Chinese
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			result := DetectLanguageFromLocale(tt.locale)
			if result != tt.expected {
				t.Errorf("Locale %s: expected %s, got %s", tt.locale, tt.expected, result)
			}
		})
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	languages := GetSupportedLanguages()
	
	// Should have at least Chinese and English
	if len(languages) < 2 {
		t.Errorf("Expected at least 2 languages, got %d", len(languages))
	}
	
	// Verify Chinese and English are present
	foundChinese := false
	foundEnglish := false
	
	for _, lang := range languages {
		if lang == Chinese {
			foundChinese = true
		}
		if lang == English {
			foundEnglish = true
		}
	}
	
	if !foundChinese {
		t.Error("Expected Chinese to be in supported languages")
	}
	if !foundEnglish {
		t.Error("Expected English to be in supported languages")
	}
}

func TestLoadTranslations(t *testing.T) {
	// Test loading valid translations
	testTranslations := Translations{
		ListTitle:       "测试",
		PackageLabel:    "包",
		StatusInstalled: "已安装",
	}
	
	err := LoadTranslations(Language("test"), testTranslations)
	if err != nil {
		t.Errorf("Expected to load valid translations, got error: %v", err)
	}
	
	// Verify translations were loaded
	translationsMu.RLock()
	_, exists := translations[Language("test")]
	translationsMu.RUnlock()
	
	if !exists {
		t.Error("Expected test language to be loaded")
	}
}

func TestLoadTranslationsInvalid(t *testing.T) {
	// Test loading translations with missing required fields
	invalidTranslations := Translations{
		ListTitle: "", // Required field missing
	}
	
	err := LoadTranslations(Language("invalid"), invalidTranslations)
	if err == nil {
		t.Error("Expected error for invalid translations, got nil")
	}
}

func TestValidateTranslations(t *testing.T) {
	errors := ValidateTranslations()
	
	// Default translations should be valid
	// Check that Chinese and English have no errors
	if zhErrors, exists := errors[Chinese]; exists {
		t.Errorf("Chinese translations should be valid, got errors: %v", zhErrors)
	}
	if enErrors, exists := errors[English]; exists {
		t.Errorf("English translations should be valid, got errors: %v", enErrors)
	}
}

func TestT(t *testing.T) {
	// Test that T returns non-empty translations
	trans := T()
	
	if trans.ListTitle == "" {
		t.Error("Expected ListTitle to be non-empty")
	}
	if trans.PackageLabel == "" {
		t.Error("Expected PackageLabel to be non-empty")
	}
}

func TestSetLanguage(t *testing.T) {
	// Save current language
	originalLang := GetLanguage()
	defer SetLanguage(originalLang)
	
	// Set to English
	SetLanguage(English)
	if GetLanguage() != English {
		t.Error("Expected language to be English")
	}
	
	// Set to Chinese
	SetLanguage(Chinese)
	if GetLanguage() != Chinese {
		t.Error("Expected language to be Chinese")
	}
}
