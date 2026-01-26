package rules

import "github.com/moby/buildkit/frontend/dockerfile/parser"

// Position represents a single point in a source file.
type Position struct {
	// Line is the 1-based line number.
	Line int `json:"line"`
	// Column is the 1-based column number (0 means column is unknown).
	Column int `json:"column,omitempty"`
}

// Location represents a range in a source file.
type Location struct {
	// File is the path to the source file.
	File string `json:"file"`
	// Start is the starting position (inclusive).
	Start Position `json:"start"`
	// End is the ending position (inclusive). If zero, it's a point location.
	End Position `json:"end"`
}

// NewFileLocation creates a location for file-level issues (no specific line).
func NewFileLocation(file string) Location {
	return Location{
		File:  file,
		Start: Position{Line: 0, Column: 0},
	}
}

// NewLineLocation creates a location for a specific line.
func NewLineLocation(file string, line int) Location {
	return Location{
		File:  file,
		Start: Position{Line: line, Column: 0},
	}
}

// NewRangeLocation creates a location spanning multiple lines/columns.
func NewRangeLocation(file string, startLine, startCol, endLine, endCol int) Location {
	return Location{
		File:  file,
		Start: Position{Line: startLine, Column: startCol},
		End:   Position{Line: endLine, Column: endCol},
	}
}

// NewLocationFromRange converts a BuildKit parser.Range to our Location type.
// This bridges BuildKit's internal types with our output schema.
func NewLocationFromRange(file string, r parser.Range) Location {
	return Location{
		File:  file,
		Start: Position{Line: r.Start.Line, Column: r.Start.Character},
		End:   Position{Line: r.End.Line, Column: r.End.Character},
	}
}

// IsFileLevel returns true if this is a file-level location (no specific line).
func (l Location) IsFileLevel() bool {
	return l.Start.Line == 0
}

// IsPointLocation returns true if this is a single-point location (no range).
func (l Location) IsPointLocation() bool {
	return l.End.Line == 0 && l.End.Column == 0
}
