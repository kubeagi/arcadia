# the base image is built from Dockerfile.vllm.ray
FROM vllm/vllm-openai:ray-2.11.0-py3.10.12-patched

# Copy the packaged python application
COPY llm-serving-app.zip /vllm-workspace/
