# Copyright 2024 KubeAGI.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from typing import Union

import pandas as pd
from datasets import Dataset
from langchain.chat_models import ChatOpenAI
from ragas import evaluate
from ragas.embeddings import RagasEmbeddings
from ragas.llms import LangchainLLM, RagasLLM
from ragas.metrics import (AnswerCorrectness, AnswerRelevancy,
                           AnswerSimilarity, ContextPrecision, ContextRecall,
                           ContextRelevancy, Faithfulness)
from ragas.metrics.base import Metric
from ragas.utils import NO_KEY
from ragas_once.embeddings.openai import OpenAIEmbeddings


class RagasEval:
    """
    The RagasOnce class is a tool for evaluating natural language models (NLMs) using various metrics for question-answering tasks. It utilizes OpenAI's language model (LLM) and text embedding models for evaluation.

    The class features include:

    Initialization: The class constructor allows customization of the API base URL, API key, LLM model, and embedding model.
    Dataset Preparation: The prepare_dataset method prepares the dataset for evaluation, splitting specific columns and converting them into lists.
    Metrics Configuration: The get_ragas_metrics method sets the metrics for evaluation, creating instances of various metric classes based on the requested metrics.
    Evaluation: The evaluate method performs the evaluation of the dataset using specified metrics, calculating summary scores and saving the results to CSV files.
    """

    # use openai llm&embedding by default
    api_base: str = "https://api.openai.com/v1"
    api_key: str = "fake"
    llm_model: str = "gpt-3.5-turbo"
    embedding_model: str = "text-embedding-ada-002"

    llm: RagasLLM
    embeddings: RagasEmbeddings

    def __init__(
        self,
        api_base: str = NO_KEY,
        api_key: str = NO_KEY,
        llm_model: str = NO_KEY,
        embedding_model: str = NO_KEY,
    ):
        # Initialize attributes based on provided arguments or default values
        self.api_base = api_base if api_base != NO_KEY else self.api_base
        self.api_key = api_key if api_key != NO_KEY else self.api_key
        self.llm_model = llm_model if llm_model != NO_KEY else self.llm_model
        self.embedding_model = (
            embedding_model if embedding_model != NO_KEY else self.embedding_model
        )

        # Initialize judge llm
        self.llm = LangchainLLM(
            llm=ChatOpenAI(
                model_name=self.llm_model,
                openai_api_key=self.api_key,
                openai_api_base=self.api_base,
            )
        )

        # Initialize judge embedding
        self.embeddings = OpenAIEmbeddings(
            api_key=self.api_key,
            api_base=self.api_base,
            model_name=self.embedding_model,
        )

    def prepare_dataset(self, dataset: str = NO_KEY) -> Dataset:
        """
        Prepares the dataset for evaluation.

        Parameters:
            dataset (str): The path to the dataset file.

        Returns:
            Dataset: The prepared dataset.

        Raises:
            ValueError: If no dataset is provided.
        """
        if dataset == NO_KEY:
            raise ValueError("No dataset provided")
        try:
            data = pd.read_csv(dataset)
        except Exception as e:
            print("An error occurred during prepare dataset:", str(e))
            return

        columns_to_split = ["ground_truths", "contexts"]
        for column in columns_to_split:
            if column in data.columns:
                data[column] = (
                    data[column].astype(str).apply(lambda x: x.split(";")).to_list()
                )

        return Dataset.from_pandas(data)

    def get_ragas_metrics(
        self, metrics: list[str], batch_size: Union[int, None] = 1
    ) -> list[Metric]:
        """
        Sets the metrics for evaluation.

        Parameters:
            metrics (list[str]): A list of metric names to be set.

        Returns:
            list[Metric]: A list of Metric objects representing the set metrics.
        """
        context_precision = ContextPrecision(llm=self.llm, batch_size=batch_size)
        context_recall = ContextRecall(llm=self.llm, batch_size=batch_size)
        context_relevancy = ContextRelevancy(llm=self.llm, batch_size=batch_size)

        answer_relevancy = AnswerRelevancy(
            llm=self.llm, embeddings=self.embeddings, batch_size=batch_size
        )
        answer_similarity = AnswerSimilarity(
            llm=self.llm, embeddings=self.embeddings, batch_size=batch_size
        )
        answer_correctness = AnswerCorrectness(
            llm=self.llm, answer_similarity=answer_similarity, batch_size=batch_size
        )
        faithfulness = Faithfulness(llm=self.llm, batch_size=batch_size)

        ms = []
        for m in metrics:
            if m == "context_precision":
                ms.append(context_precision)
            elif m == "context_recall":
                ms.append(context_recall)
            elif m == "context_relevancy":
                ms.append(context_relevancy)
            elif m == "answer_relevancy":
                ms.append(answer_relevancy)
            elif m == "answer_correctness":
                ms.append(answer_correctness)
            elif m == "answer_similarity":
                ms.append(answer_similarity)
            elif m == "faithfulness":
                ms.append(faithfulness)
        return ms

    def evaluate(
        self,
        dataset: Dataset,
        metrics: list[Metric] | None = None,
        column_map: dict[str, str] = {},
    ):
        """
        Evaluates the dataset using the specified metrics and saves the evaluation results.

        Parameters:
            dataset (Dataset): The dataset to be evaluated.
            metrics (list[Metric] | None): The list of metrics to evaluate the dataset. Defaults to None.
            column_map (dict[str, str]): A mapping of column names in the dataset to the corresponding column names
                                        expected by the evaluation function. Defaults to an empty dictionary.

        Returns:
            None
        """
        try:
            result = evaluate(dataset, metrics, column_map)
        except Exception as e:
            print("An error occurred during evaluation:", str(e))
            return
        # count total score and avearge
        summary = result.scores.to_pandas().mean()
        summary["total_score"] = summary.mean()
        summary.to_csv("summary.csv")
        result.to_pandas().to_csv("result.csv")
