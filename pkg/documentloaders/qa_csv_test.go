package documentloaders

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVLoader(t *testing.T) {
	t.Parallel()
	fileName := "./testdata/qa.csv"
	file, err := os.Open(fileName)
	assert.NoError(t, err)

	loader := NewQACSV(file, fileName)

	docs, err := loader.Load(context.Background())
	require.NoError(t, err)
	require.Len(t, docs, 9)

	expectedFileName := "员工考勤管理制度-2023.pdf"
	expected1PageContent := "q: 员工在病假期间，是否有额外的假期？"
	assert.Equal(t, docs[0].PageContent, expected1PageContent)

	expected1Metadata := map[string]any{
		AnswerCol:       "无法确定，题目未提及。",
		QAFileName:      fileName,
		LineNumber:      "0",
		FileNameCol:     expectedFileName,
		ChunkContentCol: "3员工请假时间小于等于2天，由直接上级、部门负责人审批，人力资源部备案；员工请假时间大于等于3天，依次由直接上级、部门负责人、公司管理层审批，人力资源部备案。二．事假1、申请事假须至少提前1天在钉钉上发起请假申请，经直属领导逐级审批通过后，抄送人力资源部备案。事假最小计算单位为0.5天，不足0.5天以0.5天计算，以此类推。2、如遇特殊情况未能事前申请，须于当日10：00前电话或其他有效方式告知直属上级和人力资源部，且在事后1日内在钉钉补充完成事假申请审批手续。3、事假理由不充分或有碍工作进度，公司可不予准假。一年内累计不能超过20天。事假扣除事假相应天数工资，期间无其他奖金、福利和补助。三．病假1、因病不能正常上班，需病假者。病假申请（急诊、门诊）审批要求和流程同事假审批，2天以上（含2天）病假需提供医院有效的病假证明。2、一年享有3天带薪病假。若3＜正常病假天数≤60，日工资按合同工资50%计算；正常病假累计天数>",
		PageNumberCol:   "3",
	}
	assert.Equal(t, docs[0].Metadata, expected1Metadata)

	expected2PageContent := "q: 公司的考勤管理制度适用于哪些人员？"
	assert.Equal(t, docs[1].PageContent, expected2PageContent)

	expected2Metadata := map[string]any{
		AnswerCol:       "公司全体正式员工及实习生。",
		QAFileName:      fileName,
		LineNumber:      "1",
		FileNameCol:     expectedFileName,
		ChunkContentCol: "1第一章总则一、目的为了严格工作纪律、提高工作效率，规范公司考勤管理，为公司考勤管理提供明确依据，现根据国家及当地地区相关法律法规，特制定本制度。二、使用范围1、本制度适用公司全体正式员工及实习生。2、员工应严格遵守工作律及考勤规章制度。各部门负责人在权限范围内有审批部门员工考勤记录的权利和严肃考勤纪律的义务，并以身作则，规范执行。3、人力资源部负责考勤信息的记录、汇总，监督考勤制度的执行。第二章工时制度及考勤方式1、考勤时间：1）公司执行五天弹性工作制，上班时间为9：00-9：30，下班时间为18：00-18：30，中午12：00-13：00为午休时间，不计入工作时间；每天工作时间不少于8小时。2）公司考虑交通通勤情况，每天上班给予10分钟延迟；9：40后为迟到打卡，每月最多迟到3次（不晚于10：00），超出则视为旷工；晚于10：00打卡且无正当理由，视为旷工半天；3）因工作原因下班晚走2小时，第二天打卡时间不晚于上午10：00，考勤打卡数据将作为员工日常管理和薪资核算的重要依据。",
		PageNumberCol:   "1",
	}
	assert.Equal(t, docs[1].Metadata, expected2Metadata)
	t.Logf("last doc question:%s", docs[len(docs)-1].PageContent)
}
