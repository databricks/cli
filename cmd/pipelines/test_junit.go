package pipelines

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Errors   int             `xml:"errors,attr"`
	Skipped  int             `xml:"skipped,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	ClassName string       `xml:"classname,attr,omitempty"`
	Name      string       `xml:"name,attr"`
	Time      string       `xml:"time,attr,omitempty"`
	Failure   *junitResult `xml:"failure"`
	Error     *junitResult `xml:"error"`
	Skipped   *junitResult `xml:"skipped"`
}

type junitResult struct {
	Message string `xml:"message,attr,omitempty"`
	Text    string `xml:",chardata"`
}

// junitCaseName splits a pytest test ID the way pytest's JUnit report does: the classname
// is the module path plus any class names, joined by dots.
func junitCaseName(testID string) (string, string) {
	parts := strings.Split(testID, "::")
	module := strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(parts[0], "/"), ".py"), "/", ".")
	if len(parts) == 1 {
		return "", module
	}
	classes := append([]string{module}, parts[1:len(parts)-1]...)
	return strings.Join(classes, "."), parts[len(parts)-1]
}

func newJUnitResult(text string) *junitResult {
	message, _, _ := strings.Cut(text, "\n")
	return &junitResult{Message: message, Text: text}
}

// junitNoResult is the error message of a collected case that has no final outcome.
const junitNoResult = "no result received"

// newJUnitReport has one entry per collected case; a case without a final outcome is an error,
// because CI tools treat skipped entries as passing.
// A non-empty problem, such as an incomplete run, adds a failing entry for the update so the
// report never passes by omission.
func newJUnitReport(suiteName, updateID string, run *testRun, problem string) junitTestSuites {
	suite := junitTestSuite{Name: suiteName}
	for _, c := range run.cases {
		entry := junitTestCase{}
		entry.ClassName, entry.Name = junitCaseName(c.id)
		if d, ok := c.duration(); ok {
			entry.Time = fmt.Sprintf("%.3f", d.Seconds())
		}

		details := ""
		if c.errorDetail != nil {
			details = errorDetailText(c.errorDetail)
		}
		if c.detailsTruncated {
			details = strings.TrimPrefix(details+"\n(details truncated)", "\n")
		}
		switch c.status {
		case testCasePassed:
		case testCaseFailed:
			entry.Failure = newJUnitResult(details)
			suite.Failures++
		case testCaseError:
			entry.Error = newJUnitResult(details)
			suite.Errors++
		case testCaseSkipped:
			entry.Skipped = &junitResult{}
			suite.Skipped++
		default:
			entry.Error = &junitResult{Message: junitNoResult}
			suite.Errors++
		}
		suite.Cases = append(suite.Cases, entry)
	}

	if problem != "" {
		suite.Cases = append(suite.Cases, junitTestCase{
			Name:  "update " + updateID,
			Error: newJUnitResult(problem),
		})
		suite.Errors++
	}
	suite.Tests = len(suite.Cases)
	return junitTestSuites{Suites: []junitTestSuite{suite}}
}

// writeJUnitReport replaces the file at path with the report.
func writeJUnitReport(path string, report junitTestSuites) error {
	data, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write JUnit report: %w", err)
	}
	return nil
}
