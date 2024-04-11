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
package rag

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	pkgconfig "github.com/kubeagi/arcadia/pkg/config"
)

const (
	totalScore = "total_score"

	csvLatencyField      = "latency"
	csvQuestionField     = "question"
	csvGroundTruthsField = "ground_truths"
	csvAnswerField       = "answer"
	csvContextsField     = "contexts"

	epsilon = 1e-6
	// TODO: support for color change via env
	// blueColorEnv = "BLUE_ENV"
	// blue         = "blue" // 散点图的颜色

	orangeEnv = "ORANGE_RNV"
	orange    = "orange" // 差

	greenEnv = "GREEN_ENV"
	green    = "green" // 好

	summarySuggestionTemplate = `通过此次评估，您的智能体得分偏低，主要体现在 <strong>%s</strong> 这 %d 项指标得分偏低。
<br>
<strong>建议您对特定场景应用的模型进行模型精调；%s。</strong>`
	noSuggestionTempalte = `通过此次评估，您的 RAG 方案得分 <span style="color:green">%.2f</span>`
)

var (
	// NOTE: if other fields are added in the Generate Test Data section, they need to be updated here as well.
	csvBasicFields = map[string]struct{}{
		csvQuestionField:     {},
		csvGroundTruthsField: {},
		csvAnswerField:       {},
		csvContextsField:     {},
		csvLatencyField:      {},
	}

	metricChinese = map[string]string{
		string(v1alpha1.AnswerRelevancy):   "答案相关度",
		string(v1alpha1.AnswerSimilarity):  "答案相似度",
		string(v1alpha1.AnswerCorrectness): "答案正确性",
		string(v1alpha1.Faithfulness):      "忠实度",
		string(v1alpha1.ContextPrecision):  "知识库精度",
		string(v1alpha1.ContextRelevancy):  "知识库相关度",
		string(v1alpha1.ContextRecall):     "知识库召回率",
		string(v1alpha1.AspectCritique):    "暂时没用到",
	}

	suggestionChinese = map[string]string{
		string(v1alpha1.AnswerRelevancy):   "调整 Embedding 模型",
		string(v1alpha1.AnswerSimilarity):  "调整 Embedding 模型",
		string(v1alpha1.AnswerCorrectness): "调整模型配置或更换模型",
		string(v1alpha1.Faithfulness):      "调整模型配置或更换模型",
		string(v1alpha1.ContextPrecision):  "调整 Embedding 模型",
		string(v1alpha1.ContextRelevancy):  "调整 Embedding 模型",
		string(v1alpha1.ContextRecall):     "调整 QA 数据",
		string(v1alpha1.AspectCritique):    "暂时没用到",
	}
)

type (
	RadarData struct {
		Type  string  `json:"type"`
		Value float64 `json:"value"`
		Color string  `json:"color"`
	}

	TotalScoreData struct {
		Score float64 `json:"score"`
		Color string  `json:"color"`
	}

	ScatterData struct {
		Score    float64 `json:"score"`
		CostTime float64 `json:"costTime"`
	}

	Report struct {
		RadarChart []RadarData    `json:"radarChart"`
		TotalScore TotalScoreData `json:"totalScore"`

		// TODO
		Summary string `json:"summary"`
	}

	// 忠实度、答案相关度、答案语义相似度、答案正确性、知识库相关度、知识库精度、知识库相似度
	// question,ground_truths,answer,contexts,latency
	ReportLine struct {
		Question     string             `json:"question"`
		GroundTruths []string           `json:"groundTruths"`
		Answer       string             `json:"answer"`
		Contexts     []string           `json:"contexts"`
		Data         map[string]float64 `json:"data"`
		TotalScore   float64            `json:"totalScore"`
		CostTime     float64            `json:"costTime"`
	}
	ReportDetail struct {
		Data  []ReportLine `json:"data"`
		Total int          `json:"total"`
	}
)

func ParseSummary(
	ctx context.Context,
	appName, ragName, namespace string,
	metricThresholds map[string]float64,
) (Report, error) {
	source, err := pkgconfig.GetSystemDatasourceOSS(ctx)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		return Report{}, err
	}

	filePath := fmt.Sprintf("evals/%s/%s/summary.csv", appName, ragName)
	object, err := source.Client.GetObject(ctx, namespace, filePath, minio.GetObjectOptions{})
	if err != nil {
		klog.Errorf("failed to get summary.csv file error %s", err)
		return Report{}, err
	}
	csvReader := csv.NewReader(object)
	report := Report{TotalScore: TotalScoreData{}, RadarChart: []RadarData{}}
	radarChecker := make(map[string]int)

	changeTotalScoreColor := false

	metrics := make([]string, 0)
	metricSuggesstion := make([]string, 0)

	// skip the first line
	firstLine := true
	for {
		line, err := csvReader.Read()
		if err != nil {
			if err != io.EOF {
				return Report{}, err
			}
			break
		}
		if firstLine {
			firstLine = false
			continue
		}
		if len(line) != 2 {
			return Report{}, fmt.Errorf("the summary file should only have two columns")
		}
		score, err := strconv.ParseFloat(line[1], 64)
		if err != nil {
			klog.Errorf("failed to parse thresholds for indicator %s, source value %s", line[0], line[1])
			return Report{}, err
		}
		if line[0] == totalScore {
			report.TotalScore = TotalScoreData{Score: score, Color: green}
			continue
		}
		nextRadarIndex := len(report.RadarChart)
		idx, ok := radarChecker[line[0]]
		if !ok {
			radarChecker[line[0]] = nextRadarIndex
			idx = nextRadarIndex
			report.RadarChart = append(report.RadarChart, RadarData{Type: line[0]})
		}
		report.RadarChart[idx].Value = score
		report.RadarChart[idx].Color = green
		if threshold, ok := metricThresholds[line[0]]; ok && score < threshold {
			report.RadarChart[idx].Color = orange
			metrics = append(metrics, metricChinese[line[0]])
			metricSuggesstion = append(metricSuggesstion, suggestionChinese[line[0]])
			changeTotalScoreColor = true
		}
	}

	if changeTotalScoreColor {
		report.TotalScore.Color = orange
		report.Summary = fmt.Sprintf(summarySuggestionTemplate, strings.Join(metrics, "、"), len(metrics), strings.Join(metricSuggesstion, "、"))
	} else {
		report.Summary = fmt.Sprintf(noSuggestionTempalte, report.TotalScore.Score*100.0)
	}
	return report, nil
}

func PraseScatterChart(ctx context.Context, appName, ragName, namespace string) ([]ScatterData, error) {
	source, err := pkgconfig.GetSystemDatasourceOSS(ctx)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		return nil, err
	}

	filePath := fmt.Sprintf("evals/%s/%s/result.csv", appName, ragName)
	object, err := source.Client.GetObject(ctx, namespace, filePath, minio.GetObjectOptions{})
	if err != nil {
		klog.Errorf("failed to get result.csv file error %s", err)
		return nil, err
	}
	csvReader := csv.NewReader(object)
	data, err := csvReader.ReadAll()
	if err != nil {
		klog.Error("failed to read csv error %s", err)
		return nil, err
	}

	extra := make([]int, 0)
	header := data[0]
	latencyIndex := 0
	for i := 1; i < len(header); i++ {
		if header[i] == csvLatencyField {
			latencyIndex = i
			continue
		}
		if _, ok := csvBasicFields[header[i]]; !ok {
			extra = append(extra, i)
		}
	}

	result := make([]ScatterData, 0)
	if len(extra) == 0 {
		return result, nil
	}

	for _, line := range data[1:] {
		costTime, _ := strconv.ParseFloat(line[latencyIndex], 64)
		sum := float64(0)
		for _, index := range extra {
			f, _ := strconv.ParseFloat(line[index], 64)
			sum += f
		}
		score := sum / float64(len(extra))
		result = append(result, ScatterData{CostTime: costTime, Score: score})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if math.Abs(result[i].CostTime-result[j].CostTime) < epsilon {
			return result[i].Score < result[j].Score
		}
		return result[i].CostTime < result[j].CostTime
	})
	return result, nil
}

func ParseResult(
	ctx context.Context,
	page, pageSize int,
	appName, ragName, namespace, sortBy, order string,
) (ReportDetail, error) {
	source, err := pkgconfig.GetSystemDatasourceOSS(ctx)
	if err != nil {
		klog.Errorf("failed to get system datasource error %s", err)
		return ReportDetail{}, err
	}

	filePath := fmt.Sprintf("evals/%s/%s/result.csv", appName, ragName)
	object, err := source.Client.GetObject(ctx, namespace, filePath, minio.GetObjectOptions{})
	if err != nil {
		klog.Errorf("failed to get result.csv file error %s", err)
		return ReportDetail{}, err
	}
	csvReader := csv.NewReader(object)

	data, err := csvReader.ReadAll()
	if err != nil {
		klog.Error("failed to read csv error %s", err)
		return ReportDetail{}, err
	}
	if len(data) == 0 {
		klog.Error("this may not be a normal csv file with one line of data in it: %s", filePath)
		return ReportDetail{}, nil
	}
	if len(data) == 1 {
		klog.Error("there's only one header row. %s", filePath)
		return ReportDetail{}, nil
	}

	extra := make([]int, 0)
	result := make([]ReportLine, len(data)-1)
	csvBasicFieldIndies := make(map[string]int)
	header := data[0]
	for i := 1; i < len(header); i++ {
		_, ok := csvBasicFields[header[i]]
		if ok {
			csvBasicFieldIndies[header[i]] = i
			continue
		}
		extra = append(extra, i)
	}
	if len(extra) == 0 {
		return ReportDetail{}, nil
	}

	for i, line := range data[1:] {
		item := ReportLine{
			Question:     line[csvBasicFieldIndies[csvQuestionField]],
			GroundTruths: []string{line[csvBasicFieldIndies[csvGroundTruthsField]]},
			Answer:       line[csvBasicFieldIndies[csvAnswerField]],
			Contexts:     []string{line[csvBasicFieldIndies[csvContextsField]]},
			Data:         make(map[string]float64),
		}
		item.CostTime, _ = strconv.ParseFloat(line[csvBasicFieldIndies[csvLatencyField]], 64)

		sum := float64(0)
		for _, idx := range extra {
			f, _ := strconv.ParseFloat(line[idx], 64)
			item.Data[header[idx]] = f
			sum += f
		}
		item.TotalScore = sum / float64(len(extra))
		result[i] = item
	}

	start, end := common.PagePosition(page, pageSize, len(data)-1)
	if sortBy != "" {
		if _, ok := result[0].Data[sortBy]; ok {
			sort.Slice(result, func(i, j int) bool {
				if order == "desc" {
					return result[i].Data[sortBy] > result[j].Data[sortBy]
				}
				return result[i].Data[sortBy] < result[j].Data[sortBy]
			})
		}
	}
	result = result[start:end]
	return ReportDetail{Data: result, Total: len(data) - 1}, nil
}
