package agent

type QueryFeatures struct {
	WordCount       int
	HasQuestionMark bool
	StartsWithWH    bool
	HasCodeTerms    bool
	HasMathTerms    bool
	HasActionVerbs  bool
	HasMultiPart    bool
	HasConstraints  bool
	HasComparison   bool
	IsFollowUp      bool
	HasCodeBlock    bool
	HasURL          bool
	OriginalQuery   string
}

type IntentCategory int

const (
	Greeting IntentCategory = iota
	Factual
	Calculation
	Explanation
	CodeGeneration
	Debugging
	Planning
	MetaQuery
)

func (ic IntentCategory) String() string {
	switch ic {
	case Greeting:
		return "Greeting"
	case Factual:
		return "Factual"
	case Calculation:
		return "Calculation"
	case Explanation:
		return "Explanation"
	case CodeGeneration:
		return "CodeGeneration"
	case Debugging:
		return "Debugging"
	case Planning:
		return "Planning"
	case MetaQuery:
		return "MetaQuery"
	default:
		return "Unknown"
	}
}

type IntentScore struct {
	Complexity     float64
	Certainty      float64
	Category       IntentCategory
	ReasoningChain []string
}

type RoutingDecision struct {
	Handler           string
	Template          string
	Provider          string
	Fallback          string
	Confidence        float64
	Category          IntentCategory
	Reasoning         []string
	SpecialistAgentID string // bridges category → specialist agent
}
