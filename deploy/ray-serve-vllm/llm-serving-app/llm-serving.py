import json
import logging
from typing import AsyncGenerator

import ray
import fastapi
# from huggingface_hub import login
from ray import serve

from fastapi import Request
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, Response, StreamingResponse

import vllm
from vllm.engine.arg_utils import AsyncEngineArgs
from vllm.engine.async_llm_engine import AsyncLLMEngine
from vllm.entrypoints.openai.cli_args import make_arg_parser
from vllm.entrypoints.openai.protocol import (ChatCompletionRequest,
                                              CompletionRequest, ErrorResponse)
from vllm.entrypoints.openai.serving_chat import OpenAIServingChat
from vllm.entrypoints.openai.serving_completion import OpenAIServingCompletion
from vllm.logger import init_logger
from vllm.usage.usage_lib import UsageContext


TIMEOUT_KEEP_ALIVE = 5  # seconds

logger = logging.getLogger("ray.serve")

app = fastapi.FastAPI()

# Modified based on https://github.com/vllm-project/vllm/blob/v0.4.1/vllm/entrypoints/openai/api_server.py

@serve.deployment(num_replicas=1)
@serve.ingress(app)
class VLLMPredictDeployment():
    def __init__(self, **kwargs):
        """
        Construct a VLLM deployment.

        Refer to https://github.com/vllm-project/vllm/blob/main/vllm/engine/arg_utils.py
        for the full list of arguments.

        Args:
            model: name or path of the huggingface model to use
            download_dir: directory to download and load the weights,
                default to the default cache dir of huggingface.
            use_np_weights: save a numpy copy of model weights for
                faster loading. This can increase the disk usage by up to 2x.
            use_dummy_weights: use dummy values for model weights.
            dtype: data type for model weights and activations.
                The "auto" option will use FP16 precision
                for FP32 and FP16 models, and BF16 precision.
                for BF16 models.
            seed: random seed.
            worker_use_ray: use Ray for distributed serving, will be
                automatically set when using more than 1 GPU
            pipeline_parallel_size: number of pipeline stages.
            tensor_parallel_size: number of tensor parallel replicas.
            block_size: token block size.
            swap_space: CPU swap space size (GiB) per GPU.
            gpu_memory_utilization: the percentage of GPU memory to be used for
                the model executor
            max_num_batched_tokens: maximum number of batched tokens per iteration
            max_num_seqs: maximum number of sequences per iteration.
            disable_log_stats: disable logging statistics.
            engine_use_ray: use Ray to start the LLM engine in a separate
                process as the server process.
            disable_log_requests: disable logging requests.
        """
        kwargs = {**kwargs, 'tensor_parallel_size': 1, 'gpu_memory_utilization': 0.9, 'model': '/data/models/qwen1.5-7b-chat', 'trust_remote_code': 'true', 'worker_use_ray': 'true', 'max_model_len': 6000}

        logger.info(f"vLLM API server version {vllm.__version__}")
        logger.info(f"kwargs: {kwargs}")

        args = AsyncEngineArgs(**kwargs)
        logger.info(f"args: {args}")
        served_model = args.model
        engine_args = AsyncEngineArgs.from_cli_args(args)
        engine = AsyncLLMEngine.from_engine_args(
            engine_args, usage_context=UsageContext.OPENAI_API_SERVER)
        args.response_role = ""
        args.lora_modules = ""
        args.chat_template = "./templates/chat-template-qwen.jinja2"
        self.openai_serving_chat = OpenAIServingChat(engine, served_model,
                                            args.response_role,
                                            args.lora_modules,
                                            args.chat_template)
        self.openai_serving_completion = OpenAIServingCompletion(
            engine, served_model, args.lora_modules)


    @app.get("/health")
    async def health(self) -> Response:
        """Health check."""
        await self.openai_serving_chat.engine.check_health()
        return Response(status_code=200)


    @app.get("/v1/models")
    async def show_available_models(self):
        models = await self.openai_serving_chat.show_available_models()
        return JSONResponse(content=models.model_dump())


    @app.get("/version")
    async def show_version(self):
        ver = {"version": vllm.__version__}
        return JSONResponse(content=ver)


    @app.post("/v1/chat/completions")
    async def create_chat_completion(self, request: ChatCompletionRequest,
                                     raw_request: Request):
        generator = await self.openai_serving_chat.create_chat_completion(
            request, raw_request)
        if isinstance(generator, ErrorResponse):
            return JSONResponse(content=generator.model_dump(),
                                status_code=generator.code)
        if request.stream:
            return StreamingResponse(content=generator,
                                     media_type="text/event-stream")
        else:
            return JSONResponse(content=generator.model_dump())


    @app.post("/v1/completions")
    async def create_completion(self, request: CompletionRequest, raw_request: Request):
        generator = await self.openai_serving_completion.create_completion(
            request, raw_request)
        if isinstance(generator, ErrorResponse):
            return JSONResponse(content=generator.model_dump(),
                                status_code=generator.code)
        if request.stream:
            return StreamingResponse(content=generator,
                                     media_type="text/event-stream")
        else:
            return JSONResponse(content=generator.model_dump())

deployment = VLLMPredictDeployment.bind()