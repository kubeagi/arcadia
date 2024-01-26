import argparse

import pandas as pd
import ragas_once.wrapper as pkg
from datasets import Dataset, load_dataset
from ragas import evaluate

def main():
    """
    This function is the entry point for the program. It parses command line arguments and sets up the necessary variables for evaluation.

    Parameters:
        None

    Returns:
        The result of evaluating the test set using the specified metrics.
    """
    parser = argparse.ArgumentParser(description="RAGAS CLI")
    parser.add_argument(
        "--model",
        type=str,
        default="gpt-3.5-turbo",
        help="Specifies the model to use for evaluation. Defaults to gpt-3.5-turbo.",
    )
    parser.add_argument(
        "--apibase",
        type=str,
        default="https://api.openai.com/v1",
        help="Specifies the base URL for the API. Defaults to OpenAI.",
    )
    parser.add_argument(
        "--apikey", type=str, help="Specifies the API key to authenticate requests."
    )
    parser.add_argument(
        "--embeddings",
        type=str,
        help="Specifies Huggingface embeddings model (or its path) to use for evaluation. Will use OpenAI embeddings if not set.",
    )
    parser.add_argument(
        "--metrics",
        type=list,
        default=[],
        help="Specifies the metrics to use for evaluation.",
    )
    parser.add_argument(
        "--dataset",
        type=str,
        help="Specifies the path to the dataset for evaluation. Will use fiqa dataset if not set.",
    )

    args = parser.parse_args()
    model = args.model
    api_base = args.apibase
    api_key = args.apikey
    metrics = args.metrics
    dataset = args.dataset

    judge_model = pkg.wrap_langchain_llm(model, api_base, api_key)

    embeddings_model_name = args.embeddings

    if embeddings_model_name:
        embeddings = pkg.wrap_embeddings("huggingface", embeddings_model_name, None)
    else:
        embeddings = pkg.wrap_embeddings("openai", None, api_key)

    if dataset:
        data = pd.read_csv(dataset)
        data["ground_truths"] = data["ground_truths"].apply(lambda x: x.split(";"))
        data["contexts"] = data["contexts"].apply(lambda x: x.split(";"))
        test_set = Dataset.from_pandas(data)
    else:
        print("test_set not provided, using fiqa dataset")
        fiqa = load_dataset("explodinggradients/fiqa", "ragas_eval")
        test_set = fiqa["baseline"].select(range(5))


    ms = pkg.set_metrics(metrics, judge_model, embeddings)

    result = evaluate(test_set, ms)
    print(result)
    
    # count total score and avearge
    summary = result.scores.to_pandas().mean()
    summary["total_score"] = summary.mean()
    summary.to_csv("summary.csv")
    result.to_pandas().to_csv("result.csv")


if __name__ == "__main__":
    main()
