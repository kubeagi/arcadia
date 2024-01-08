from ragas import evaluate
from datasets import Dataset
from ragas.llms import LangchainLLM
from langchain.chat_models import ChatOpenAI
from langchain.embeddings import OpenAIEmbeddings
# create Langchain instance
chat = ChatOpenAI(
    model="a3e0c8a6-101c-4000-a1cd-d523ff7f521d",
    openai_api_key="key",
    openai_api_base="http://fastchat-api.172.22.96.167.nip.io/v1",
    max_tokens=100,
    temperature=0.5,
)

embedding = OpenAIEmbeddings(
    model="a3e0c8a6-101c-4000-a1cd-d523ff7f521d",
    openai_api_key="key",
    openai_api_base="http://fastchat-api.172.22.96.167.nip.io/v1"
)

# use the Ragas LangchainLLM wrapper to create a RagasLLM instance
llm = LangchainLLM(llm=chat)

from ragas.metrics import ContextPrecision, ContextRecall, ContextRelevancy
from ragas.metrics import AnswerCorrectness,AnswerRelevancy,AnswerSimilarity,Faithfulness

# change the LLM

context_precision = ContextPrecision(llm=llm)
context_recall = ContextRecall(llm=llm)
context_relevancy = ContextRelevancy(llm=llm)
answer_relevancy = AnswerRelevancy(llm=llm,embeddings=embedding)
answer_similarity = AnswerSimilarity(llm=llm,embeddings=embedding,is_cross_encoder=True)
answer_correctness = AnswerCorrectness(llm=llm,answer_similarity=answer_similarity)
faithfulness = Faithfulness(llm=llm)

m = [context_precision,context_recall,context_relevancy,
    answer_relevancy,answer_correctness,faithfulness,answer_similarity]

q = [
    "部门负责人在考勤管理中有哪些权利和义务？",
    "公司对迟到打卡有哪些规定？",
    "员工请假需要提前多久申请？"
]

c1 = [
    "员工应严格遵守工作纪律并规范执行。",
    "部门负责人在权限范围内有审批部门员工考勤记录的权利和严肃考勤纪律的义务，并以身作则，规范执行。",
    "人力资源部负责考勤信息的记录、汇总和监督考勤制度的执行。"
]

c2 = [
    "公司实行五天弹性工作制，每天工作时间不少于8小时。",
    "每天上班给予10分钟延迟；9：40后为迟到打卡，每月最多迟到3次（不晚于10：00），超出则视为旷工；晚于10：00打卡且无正当理由，视为旷工半天。",
    "公司考虑交通通勤情况，每天上班给予10分钟延迟。"
]

c3 = [
    "员工请假时间小于等于2天，由直接上级、部门负责人审批，人力资源部备案；员工请假时间大于等于3天，依次由直接上级、部门负责人、公司管理层审批，人力资源部备案。",
    "员工需要提前申请请假，具体请假时间视情况而定，一般来说，提前一天或两天向直接上级或部门负责人申请请假即可。"
]

a = [
    "部门负责人在权限范围内有审批部门员工考勤记录的权利和严肃考勤纪律的义务，并以身作则，规范执行。",
    "每天上班给予10分钟延迟；9：40后为迟到打卡，每月最多迟到3次（不晚于10：00），超出则视为旷工；晚于10：00打卡且无正当理由，视为旷工半天。",
    "员工需要提前申请请假，具体请假时间视情况而定，一般来说，提前一天或两天向直接上级或部门负责人申请请假即可。"
]

c = [c1,c2,c3]
g = [c1,c2,c3]

dataset_dict = {
    'question': q,
    'answer': a,
    'contexts': c,
    'ground_truths': g
}

dataset = Dataset.from_dict(
    {
    "question": q,
    "answer": a,
    "contexts": c,
    "ground_truths": g,
    },
)
result = evaluate(dataset, metrics=m)
print(result)