package storage

import (
	"github.com/habibiefaried/email-server/internal/parser"
)

// ParsedEmail is a backward compatibility type alias for parser.Email
type ParsedEmail = parser.Email

// AttachmentData is a backward compatibility type alias for parser.Attachment
type AttachmentData = parser.Attachment

// ParseEmail is a backward compatibility function that wraps parser.Parse
// Deprecated: Use parser.Parse directly instead
func ParseEmail(rawContent string) (*ParsedEmail, error) {
	return parser.Parse(rawContent)
}
