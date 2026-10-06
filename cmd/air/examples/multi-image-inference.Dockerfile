FROM vllm/vllm-openai:v0.10.2

ARG PIP_INDEX_URL

RUN python3 -m pip install --no-cache-dir \
      'trl==0.24.0' 'datasets==4.2.0' && \
    python3 -c 'import trl, datasets, vllm; assert (trl.__version__, datasets.__version__, vllm.__version__) == ("0.24.0", "4.2.0", "0.10.2")'
