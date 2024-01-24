/*
Copyright 2024 KubeAGI.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

type MetricsKind string

const (
	// AnswerRelevancy in ragas https://docs.ragas.io/en/stable/concepts/metrics/answer_relevance.html
	// Scores the relevancy of the answer according to the given question.
	AnswerRelevancy MetricsKind = "answer_relevancy"

	// AnswerSimilarity in ragas https://docs.ragas.io/en/stable/concepts/metrics/semantic_similarity.html
	// Scores the semantic similarity of ground truth with generated answer.
	AnswerSimilarity MetricsKind = "answer_similarity"

	// AnswerCorrectness in ragas https://docs.ragas.io/en/stable/concepts/metrics/answer_correctness.html
	// Measures answer correctness compared to ground truth as a combination(Weighted) of
	// - factuality
	// - semantic similarity
	AnswerCorrectness MetricsKind = "answer_correctness"

	// Faithfulness in ragas https://docs.ragas.io/en/stable/concepts/metrics/faithfulness.html
	// Scores the factual consistency of the generated answer against the given context.
	Faithfulness MetricsKind = "faithfulness"

	// ContextPrecision in ragas https://docs.ragas.io/en/stable/concepts/metrics/context_precision.html
	// Average Precision is a metric that evaluates whether all of the relevant items selected by the model are ranked higher or not.
	ContextPrecision MetricsKind = "context_precision"

	// ContextRelevancy in ragas https://docs.ragas.io/en/stable/concepts/metrics/context_relevancy.html
	// Gauges the relevancy of the retrieved context
	ContextRelevancy MetricsKind = "context_relevancy"

	// ContextRecall in ragas https://docs.ragas.io/en/stable/concepts/metrics/context_recall.html
	// Estimates context recall by estimating TP and FN using annotated answer and retrieved context.
	ContextRecall MetricsKind = "context_recall"

	// AspectCritique in ragas https://docs.ragas.io/en/stable/concepts/metrics/critique.html
	// Designed to assess submissions based on predefined aspects such as harmlessness and correctness
	// SUPPORTED_ASPECTS = [ harmfulness, maliciousness, coherence, correctness, conciseness, ]
	AspectCritique MetricsKind = "aspect_critique"
)

type Metric struct {
	// Kind of this Metric
	Kind MetricsKind `json:"kind,omitempty"`

	// Parameters in this Metrics
	Parameters []Parameter `json:"parameters,omitempty"`

	// ToleranceThreshbold on this Metric
	// If the evaluation score is smaller than this tolerance threshold,we treat this RAG solution as `Bad`
	ToleranceThreshbold int `json:"tolerance_threshold,omitempty"`
}

// Parameter to metrics which is a key-value pair
type Parameter struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// Report is the summarization of evaluation
type Report struct {
	// TODO
}

// Define RAG common structure and variables
const (
	EvaluationJobLabels        = Group + "/rag"
	EvaluationApplicationLabel = Group + "/application"
)

func RagStatusChanged(a, b RAGStatus) bool {
	if !a.CompletionTime.Equal(b.CompletionTime) {
		return true
	}
	if a.Phase != b.Phase {
		return true
	}
	ac, bc := a.Conditions, b.Conditions
	la, lb := len(ac), len(bc)
	if la != lb {
		return true
	}
	if la == 0 {
		return false
	}
	return ac[0].Type != bc[0].Type || ac[0].Status != bc[0].Status ||
		ac[0].Reason != bc[0].Reason || ac[0].Message != bc[0].Message
}
