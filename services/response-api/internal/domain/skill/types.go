package skill

import "context"

// SkillType identifies the supported document generation skills.
type SkillType string

const (
	SkillTypeSlides       SkillType = "slides"
	SkillTypeDocs         SkillType = "docs"
	SkillTypePDFs         SkillType = "pdfs"
	SkillTypeSpreadsheets SkillType = "spreadsheets"
)

// Service defines the interface for skill-related operations.
type Service interface {
	GenerateCode(ctx context.Context, req GenerateCodeRequest) (string, error)
}

// GenerateCodeRequest contains inputs for building skill execution code.
type GenerateCodeRequest struct {
	SkillType  SkillType
	Content    interface{}
	Options    map[string]interface{}
	OutputPath string
}
