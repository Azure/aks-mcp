package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Loader struct{}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) LoadTestCase(path string) (*TestCase, error) {
	var testCasePath string
	
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if fi.IsDir() {
		testCasePath = filepath.Join(path, "test_case.yaml")
	} else {
		testCasePath = path
	}

	data, err := os.ReadFile(testCasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read test case: %w", err)
	}

	var tc TestCase
	if err := yaml.Unmarshal(data, &tc); err != nil {
		return nil, fmt.Errorf("failed to parse test case: %w", err)
	}

	tc.FilePath = filepath.Dir(testCasePath)

	if tc.Timeout == 0 {
		tc.Timeout = 300 * time.Second
	}

	if err := l.validateTestCase(&tc); err != nil {
		return nil, fmt.Errorf("invalid test case: %w", err)
	}

	return &tc, nil
}

func (l *Loader) LoadTestCases(path string) ([]*TestCase, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if !fi.IsDir() {
		tc, err := l.LoadTestCase(path)
		if err != nil {
			return nil, err
		}
		return []*TestCase{tc}, nil
	}

	testCaseFile := filepath.Join(path, "test_case.yaml")
	if _, err := os.Stat(testCaseFile); err == nil {
		tc, err := l.LoadTestCase(path)
		if err != nil {
			return nil, err
		}
		return []*TestCase{tc}, nil
	}

	var testCases []*TestCase

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCasePath := filepath.Join(path, entry.Name())
		testCaseFile := filepath.Join(testCasePath, "test_case.yaml")

		if _, err := os.Stat(testCaseFile); os.IsNotExist(err) {
			continue
		}

		tc, err := l.LoadTestCase(testCasePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load test case from %s: %v\n", testCasePath, err)
			continue
		}

		testCases = append(testCases, tc)
	}

	if len(testCases) == 0 {
		return nil, fmt.Errorf("no test cases found in %s", path)
	}

	return testCases, nil
}

func (l *Loader) validateTestCase(tc *TestCase) error {
	if tc.ID == "" {
		return fmt.Errorf("test case ID is required")
	}
	if tc.Title == "" {
		return fmt.Errorf("test case title is required")
	}
	if tc.UserPrompt == "" {
		return fmt.Errorf("user_prompt is required")
	}
	if len(tc.ExpectedOutput) == 0 {
		return fmt.Errorf("expected_output is required")
	}
	return nil
}

func (l *Loader) FilterByTags(testCases []*TestCase, tags []string) []*TestCase {
	if len(tags) == 0 {
		return testCases
	}

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	var filtered []*TestCase
	for _, tc := range testCases {
		for _, tcTag := range tc.Tags {
			if tagSet[tcTag] {
				filtered = append(filtered, tc)
				break
			}
		}
	}

	return filtered
}
