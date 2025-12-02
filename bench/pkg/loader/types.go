package loader

import "time"

// TestCase represents a single benchmark test case
type TestCase struct {
	// Metadata
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`

	// Test content
	UserPrompt     string   `yaml:"user_prompt"`
	ExpectedOutput []string `yaml:"expected_output"`

	// Expected tool usage for validation
	ExpectedToolUsage []ExpectedToolUsage `yaml:"expected_tool_usage"`

	// Environment setup
	BeforeTest string `yaml:"before_test"`
	AfterTest  string `yaml:"after_test"`

	// Timeout
	Timeout time.Duration `yaml:"timeout"`

	// File path
	FilePath string `yaml:"-"`
}

// ExpectedToolUsage defines the expected tool calls for validation
type ExpectedToolUsage struct {
	ToolName             string   `yaml:"tool_name"`
	MinCalls             int      `yaml:"min_calls"`
	RequiredArgsPatterns []string `yaml:"required_args_patterns"`
}

// TestResult represents the result of running a test case
type TestResult struct {
	TestCase *TestCase
	
	// Execution info
	StartTime     time.Time
	EndTime       time.Time
	ExecutionTime time.Duration
	
	// Performance metrics
	TotalToolCallTime   time.Duration
	AvgToolCallDuration time.Duration
	LLMIterations       int
	
	// Agent output
	FinalAnswer string
	ToolCalls   []ToolCallRecord
	
	// Evaluation results
	ToolSelectionCorrect bool
	ToolSelectionScore   float64
	ParameterAccuracy    float64
	OutputQualityScore   float64
	
	// Status
	Status       TestStatus
	ErrorMessage string
	
	// Detailed validation
	ValidationResults []*ToolValidationResult
	JudgingResult     *JudgingResult
}

// ToolCallRecord records a single tool call made by the agent
type ToolCallRecord struct {
	ToolName      string
	Arguments     map[string]interface{}
	Result        string
	Error         string
	Timestamp     time.Time
	ExecutionTime time.Duration
}

// ToolValidationResult contains the validation result for a tool usage expectation
type ToolValidationResult struct {
	ExpectedTool    string
	ToolCalled      bool
	CallCount       int
	MinCalls        int
	ArgsValid       bool
	MatchedPatterns []string
	MissingPatterns []string
	Score           float64
}

// JudgingResult contains the LLM judge's evaluation of the answer quality
type JudgingResult struct {
	Score     float64
	Reasoning string
}

// TestStatus represents the status of a test
type TestStatus string

const (
	TestStatusPass    TestStatus = "pass"
	TestStatusFail    TestStatus = "fail"
	TestStatusError   TestStatus = "error"
	TestStatusSkipped TestStatus = "skipped"
)

// BenchmarkSummary summarizes results from multiple tests
type BenchmarkSummary struct {
	TotalTests             int
	PassedTests            int
	FailedTests            int
	ErrorTests             int
	SkippedTests           int
	
	SuccessRate            float64
	AvgToolSelectionScore  float64
	AvgParameterAccuracy   float64
	AvgOutputQualityScore  float64
	
	TotalExecutionTime     time.Duration
	AvgTestDuration        time.Duration
	AvgToolCallDuration    time.Duration
	AvgToolCallsPerTest    float64
	AvgLLMIterations       float64
	
	TestResults            []*TestResult
	
	Timestamp              time.Time
	Model                  string
	MCPBinary              string
}
