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

import argparse

from ragas_once.eval import RagasEval


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
        "--apibase",
        type=str,
        default="https://api.openai.com/v1",
        help="Specifies the base URL for the API. Defaults to OpenAI.",
    )
    parser.add_argument(
        "--apikey", type=str, help="Specifies the API key to authenticate requests."
    )
    parser.add_argument(
        "--llm",
        type=str,
        default="gpt-3.5-turbo",
        help="Specifies the model to use for evaluation. Defaults to gpt-3.5-turbo.",
    )
    parser.add_argument(
        "--embedding",
        type=str,
        default="text-embedding-ada-002",
        help="Specifies embeddings model (or its path) to use for evaluation. Will use OpenAI embeddings if not set.",
    )
    parser.add_argument(
        "--metrics",
        type=str,
        default="answer_relevancy,context_precision,context_recall,context_relevancy,faithfulness",
        help="Specifies the metrics to use for evaluation. Comma-separated values.",
    )
    parser.add_argument(
        "--dataset",
        type=str,
        help="Specifies the path to the dataset for evaluation. Will use fiqa dataset if not set.",
    )

    args = parser.parse_args()

    # Initialize ragas_once with provided arguments
    once = RagasEval(
        api_base=args.apibase,
        api_key=args.apikey,
        llm_model=args.llm,
        embedding_model=args.embedding,
    )

    # Prepare the dataset
    dataset = once.prepare_dataset(args.dataset)

    if dataset is None:
        raise ValueError("No dataset provided")

    # Get the metrics to evaluate
    metrics = once.get_ragas_metrics(args.metrics.split(","))

    # Run the evaluation
    once.evaluate(dataset=dataset, metrics=metrics)


if __name__ == "__main__":
    main()
