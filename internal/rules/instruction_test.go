package rules

import "testing"

func TestInstructionType_String(t *testing.T) {
	tests := []struct {
		t    InstructionType
		want string
	}{
		{InstructionFROM, "FROM"},
		{InstructionRUN, "RUN"},
		{InstructionCMD, "CMD"},
		{InstructionCOPY, "COPY"},
		{InstructionADD, "ADD"},
		{InstructionENV, "ENV"},
		{InstructionARG, "ARG"},
		{InstructionNone, "(file-level)"},
		{InstructionAll, "(all)"},
		{InstructionFROM | InstructionRUN, "(multiple)"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.t.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInstructionType_Contains(t *testing.T) {
	tests := []struct {
		t, other InstructionType
		want     bool
	}{
		{InstructionFROM, InstructionFROM, true},
		{InstructionFROM, InstructionRUN, false},
		{InstructionFROM | InstructionRUN, InstructionFROM, true},
		{InstructionFROM | InstructionRUN, InstructionCOPY, false},
		{InstructionAll, InstructionFROM, true},
		{InstructionAll, InstructionRUN, true},
		{InstructionNone, InstructionFROM, false},
	}

	for _, tc := range tests {
		if got := tc.t.Contains(tc.other); got != tc.want {
			t.Errorf("%v.Contains(%v) = %v, want %v", tc.t, tc.other, got, tc.want)
		}
	}
}

func TestParseInstructionType(t *testing.T) {
	tests := []struct {
		name string
		want InstructionType
	}{
		{"FROM", InstructionFROM},
		{"RUN", InstructionRUN},
		{"CMD", InstructionCMD},
		{"COPY", InstructionCOPY},
		{"ADD", InstructionADD},
		{"ENV", InstructionENV},
		{"ARG", InstructionARG},
		{"LABEL", InstructionLABEL},
		{"EXPOSE", InstructionEXPOSE},
		{"VOLUME", InstructionVOLUME},
		{"USER", InstructionUSER},
		{"WORKDIR", InstructionWORKDIR},
		{"ENTRYPOINT", InstructionENTRYPOINT},
		{"SHELL", InstructionSHELL},
		{"HEALTHCHECK", InstructionHEALTHCHECK},
		{"STOPSIGNAL", InstructionSTOPSIGNAL},
		{"ONBUILD", InstructionONBUILD},
		{"MAINTAINER", InstructionMAINTAINER},
		{"unknown", InstructionNone},
		{"", InstructionNone},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseInstructionType(tc.name); got != tc.want {
				t.Errorf("ParseInstructionType(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
