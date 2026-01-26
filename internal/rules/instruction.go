package rules

// InstructionType represents a Dockerfile instruction type.
// Used as a bitset for dispatch hints.
type InstructionType uint32

const (
	// Individual instruction types
	InstructionFROM InstructionType = 1 << iota
	InstructionRUN
	InstructionCMD
	InstructionLABEL
	InstructionMAINTAINER
	InstructionEXPOSE
	InstructionENV
	InstructionADD
	InstructionCOPY
	InstructionENTRYPOINT
	InstructionVOLUME
	InstructionUSER
	InstructionWORKDIR
	InstructionARG
	InstructionONBUILD
	InstructionSTOPSIGNAL
	InstructionHEALTHCHECK
	InstructionSHELL

	// InstructionNone means the rule doesn't use instruction dispatch.
	// Also used to indicate file-level rules.
	InstructionNone InstructionType = 0

	// InstructionAll means the rule applies to all instructions.
	InstructionAll InstructionType = ^InstructionType(0)
)

// Contains returns true if the bitset contains the given instruction type.
func (t InstructionType) Contains(other InstructionType) bool {
	return t&other != 0
}

// String returns a human-readable name for single instruction types.
func (t InstructionType) String() string {
	switch t {
	case InstructionFROM:
		return "FROM"
	case InstructionRUN:
		return "RUN"
	case InstructionCMD:
		return "CMD"
	case InstructionLABEL:
		return "LABEL"
	case InstructionMAINTAINER:
		return "MAINTAINER"
	case InstructionEXPOSE:
		return "EXPOSE"
	case InstructionENV:
		return "ENV"
	case InstructionADD:
		return "ADD"
	case InstructionCOPY:
		return "COPY"
	case InstructionENTRYPOINT:
		return "ENTRYPOINT"
	case InstructionVOLUME:
		return "VOLUME"
	case InstructionUSER:
		return "USER"
	case InstructionWORKDIR:
		return "WORKDIR"
	case InstructionARG:
		return "ARG"
	case InstructionONBUILD:
		return "ONBUILD"
	case InstructionSTOPSIGNAL:
		return "STOPSIGNAL"
	case InstructionHEALTHCHECK:
		return "HEALTHCHECK"
	case InstructionSHELL:
		return "SHELL"
	case InstructionNone:
		return "(file-level)"
	case InstructionAll:
		return "(all)"
	default:
		return "(multiple)"
	}
}

// ParseInstructionType converts an instruction name to its type.
func ParseInstructionType(name string) InstructionType {
	switch name {
	case "FROM":
		return InstructionFROM
	case "RUN":
		return InstructionRUN
	case "CMD":
		return InstructionCMD
	case "LABEL":
		return InstructionLABEL
	case "MAINTAINER":
		return InstructionMAINTAINER
	case "EXPOSE":
		return InstructionEXPOSE
	case "ENV":
		return InstructionENV
	case "ADD":
		return InstructionADD
	case "COPY":
		return InstructionCOPY
	case "ENTRYPOINT":
		return InstructionENTRYPOINT
	case "VOLUME":
		return InstructionVOLUME
	case "USER":
		return InstructionUSER
	case "WORKDIR":
		return InstructionWORKDIR
	case "ARG":
		return InstructionARG
	case "ONBUILD":
		return InstructionONBUILD
	case "STOPSIGNAL":
		return InstructionSTOPSIGNAL
	case "HEALTHCHECK":
		return InstructionHEALTHCHECK
	case "SHELL":
		return InstructionSHELL
	default:
		return InstructionNone
	}
}
