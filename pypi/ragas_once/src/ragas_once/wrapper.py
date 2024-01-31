import os

from datasets import Dataset
from langchain.chat_models import ChatOpenAI
from ragas.embeddings import (HuggingfaceEmbeddings, OpenAIEmbeddings,
                              RagasEmbeddings)
from ragas.llms import LangchainLLM, RagasLLM
from ragas.metrics import (answer_correctness, answer_relevancy,
                           answer_similarity, context_precision,
                           context_recall, context_relevancy, faithfulness)
from ragas.metrics.base import Metric

DEFAULT_METRICS = [
    "answer_relevancy",
    "context_precision",
    "faithfulness",
    "context_recall",
    "context_relevancy",
]


def wrap_langchain_llm(
    model: str, api_base: str | None, api_key: str | None
) -> LangchainLLM:
    """
    Initializes and returns an instance of the LangchainLLM class.

    Args:
        model (str): The name of the language model to use.
        api_base (str | None): The base URL for the OpenAI API. If None, the default URL is assumed.
        api_key (str | None): The API key for the OpenAI API.

    Returns:
        LangchainLLM: An instance of the LangchainLLM class.

    Raises:
        ValueError: If api_key is not provided.

    Notes:
        - If api_base is not provided, the default URL 'https://api.openai.com/v1' is assumed.
        - The environment variables OPENAI_API_KEY and OPENAI_API_BASE are set to the provided api_key and api_base.
    """
    if api_base is None:
        print("api_base not provided, assuming OpenAI default")
        if api_key is None:
            raise ValueError("api_key must be provided")
        os.environ["OPENAI_API_KEY"] = api_key
        base = ChatOpenAI(model_name=model)
    else:
        os.environ["OPENAI_API_BASE"] = api_base
        if api_key:
            os.environ["OPENAI_API_KEY"] = api_key
        base = ChatOpenAI(
            model_name=model, openai_api_key=api_key, openai_api_base=api_base
        )
    return LangchainLLM(llm=base)


def set_metrics(
    metrics: list[str], llm: RagasLLM | None, embeddings: RagasEmbeddings | None
) -> list[Metric]:
    """
    Sets the metrics for evaluation.

    Parameters:
        metrics (list[str]): A list of metric names to be set.
        llm (RagasLLM | None): An instance of RagasLLM or None. If not set, the code will use OpenAI ChatGPT as default.
        embeddings (RagasEmbeddings | None): An instance of RagasEmbeddings or None. If not set, the code will use OpenAI Embeddings as default.

    Returns:
        list[Metric]: A list of Metric objects representing the set metrics.
    """
    ms = []
    if llm:
        context_precision.llm = llm
        context_recall.llm = llm
        context_relevancy.llm = llm
        answer_correctness.llm = llm
        answer_similarity.llm = llm
        faithfulness.llm = llm
    if embeddings:
        answer_relevancy.embeddings = embeddings
        answer_correctness.embeddings = embeddings
        answer_similarity.embeddings = embeddings
    if not metrics:
        metrics = DEFAULT_METRICS
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


def wrap_embeddings(
    model_type: str, model_name: str | None, api_key: str | None
) -> RagasEmbeddings:
    if model_type == "openai":
        return OpenAIEmbeddings(api_key=api_key)
    elif model_type == "huggingface":
        return HuggingfaceEmbeddings(model_name=model_name)
    else:
        raise ValueError(f"Invalid model type: {model_type}")
