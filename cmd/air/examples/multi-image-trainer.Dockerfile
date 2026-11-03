FROM huggingface/trl:0.23.1

ARG PIP_INDEX_URL

RUN python -m pip install --no-cache-dir \
      'trl==0.24.0' 'vllm==0.10.2' && \
    python -c 'import trl, vllm; assert (trl.__version__, vllm.__version__) == ("0.24.0", "0.10.2")'
