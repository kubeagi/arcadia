# 北京住房公积金 Bot

## 思路 1

切割文件并传到向量数据库，把用户问题和向量数据库比较，找 5 个相似片段，一块发到 LLM 去回答。类似 chat_with_document 的思路。或者 https://claude.ai/ 可以直接体验这个过程。

## 思路 2

按决策树的方式，解析政策文件。把用户的问题定位到决策树的叶节点，然后把问题和叶子节点的内容一块交给 LLM 去回答。

### 决策树的构建

解析 markdown 的标题，生成结果为(用 `-` 表示标题的级别)

```
- 1. 哪些情况可以提取住房公积金？是否可以通过网上申请？
- 2. 提取住房公积金是否必须单位办理，个人可以办理住房公积金提取业务吗？
- 购买住房如何提取住房公积金？
-- 3. 购买北京市行政区域内住房如何提取住房公积金？
--- 一、购房人申请提取本人住房公积金
---- 途径一：通过住房公积金网上业务系统申请。
---- 途径二：通过北京公积金 APP 申请
---- 途径三：通过住房公积金业务柜台办理。
--- 二、缴存人因同一套住房曾办理过购房提取，中断后可再次申请提取。
---- 途径一：通过住房公积金网上业务系统办结。
---- 途径二：通过住房公积金业务柜台办理。
-- 4. 购房时已办理了北京住房公积金贷款如何提取住房公积金？
--- 一、购房人申请提取本人住房公积金
---- 途径一：通过住房公积金网上业务系统申请，网上办结。
---- 途径二：北京公积金 APP 办理，网上办结
---- 途径三：通过住房公积金业务柜台办理。
--- 二、缴存人因同一套住房曾办理过购房提取，中断后可再次申请提取。
---- 途径一：通过住房公积金网上业务系统办结。
---- 途径二：通过住房公积金业务柜台办理。
-- 5. 全款购买北京市行政区域外住房是否可以申请提取？如何提取？
--- 一、购房人申请提取本人住房公积金
---- 途径一：通过住房公积金网上业务系统申请，全程网办。
---- 途径二：通过住房公积金业务柜台办理。
--- 二、缴存人因同一套住房曾办理过购房提取，中断后可再次申请提取。北京住房公积金缴存人因同一套住房曾办理过购房提取，提取金额未达到购房款总额，中断后可再次申请提取。
---- 途径一：通过住房公积金网上业务系统办结。
---- 途径二：通过住房公积金业务柜台办理。
-- 6. 使用商业银行贷款及使用异地住房公积金贷款购买北京市行政区域外住房是否可以申请提取？如何提取？
--- 一、购房人申请提取本人住房公积金
---- 途径一：通过住房公积金网上业务系统申请。
---- 途径二：通过住房公积金业务柜台办理。
--- 二、缴存人因同一套住房曾办理过购房提取，中断后可再次申请提取
---- 途径一：通过住房公积金网上业务系统办结。
---- 途径二：通过住房公积金业务柜台办理。
-- 7. 家庭购房应由哪一方先行办理提取住房公积金手续？如购房人一方已办理提取，其配偶如何办理提取？
-- 9. 父母购房，子女是否可以申请提取本人的住房公积金？
-- 10. 同一套住房已办理过购房提取的，未到提取限额如何再次办理住房公积金提取？
-- 11. 购买多套房屋，第一套房的提取额度用完了，如何继续提取住房公积金？
-- 13. 办理购房提取住房公积金，夫妻双方提取额度如何划分？
- 租房如何提取公积金？
-- 8. 无租房合同及发票如何提取住房公积金？
--- 途径一：通过住房公积金网上业务系统申请，网上办结。
--- 途径二：通过北京公积金 APP（或京通小程序）办理申请
--- 途径三：通过住房公积金业务柜台办理。
- 12. 已办理了租房提取，又购买了住房，是否能办理购房提取？
- 14. 退休后如何销户提取住房公积金？
-- 途径一：通过住房公积金网上业务系统申请，网上办结。
-- 途径二：通过北京公积金 APP 申请，网上办结
-- 途径三：通过住房公积金业务柜台办理。
- 15. 因继承如何提取被继承人的住房公积金？
-- 途径一：通过住房公积金网上业务系统申请，网上办结。
-- 途径二：通过住房公积金业务柜台办理。
-- 途径三：通过邮寄材料办理
- 16. 非本市户籍人员离职后是否可以办理住房公积金销户提取？
-- 途径一：通过住房公积金网上业务系统申请办理。
-- 途径二：通过住房公积金业务柜台办理。
- 约定提取
-- 17. 什么是约定提取？
-- 18. 约定提取住房公积金未到账是什么原因，如何处理？
-- 19. 为何约定提取到账金额与上次约定提取到账金额不一样？
-- 21. 已办理提取住房公积金手续，但没办理约定提取，如何在网上办理？
- 20. 使用虚假、伪造材料申请办理住房公积金提取业务的，缴存职工将面临什么后果？
- 22. 销户提取公积金包含哪些事项
```

### 用户问题找关键节点

试过用向量匹配用户问题和 LLM 查找，结果都比较一般，可能是原始文件需要更大力度的清洗和重构。

LLM 的 prompt 为：

```
请根据以下问题和标题，找出问题最符合的标题序号，然后只输出该序号，不要包含额外的文字或标点符号。

问题: <用户输入1> <用户输入2>

标题: 
<一级标题1>
<一级标题1> <二级标题11>
<一级标题1> <二级标题12> <三级标题121>
<一级标题2>
<一级标题2> <二级标题21>
<一级标题2> <二级标题22> <三级标题221>
```

### 找到叶节点，询问 LLM

prompt:

```
我将提供一些内容并提出一个问题，您应该根据我提供的内容来回答。请使用您的知识和理解来回答下列问题，如果不知道，请回复'不知道':
问题：<用户输入1> <用户输入2>
---
内容：
<叶节点标题>
<叶节点正文内容>
```

## 原始数据

raw_data 中保留了政策文件原文：

- 住房公积金提取业务问答.md 来自 [https://gjj.beijing.gov.cn/web/zwfw5/1747335/1747336/10903657/index.html](https://gjj.beijing.gov.cn/web/zwfw5/1747335/1747336/10903657/index.html) 为 2023-10-11 的版本，未做任何修改。