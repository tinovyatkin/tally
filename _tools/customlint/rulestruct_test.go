package customlint

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestRuleStruct(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, ruleStructAnalyzer, "internal/rules")

}
