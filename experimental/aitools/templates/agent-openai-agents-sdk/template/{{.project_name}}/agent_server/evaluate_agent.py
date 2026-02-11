import asyncio

import mlflow
from dotenv import load_dotenv
from mlflow.genai.agent_server import get_invoke_function
from mlflow.genai.scorers import RelevanceToQuery, Safety
from mlflow.types.responses import ResponsesAgentRequest, ResponsesAgentResponse

# Load environment variables from .env.local if it exists
load_dotenv(dotenv_path=".env.local", override=True)

# Import agent for our @invoke function to be found
from agent_server import agent  # noqa: F401

# Create your evaluation dataset
# Refer to documentation for evaluations:
# Scorers: https://docs.databricks.com/aws/en/mlflow3/genai/eval-monitor/concepts/scorers
# Predefined LLM scorers: https://mlflow.org/docs/latest/genai/eval-monitor/scorers/llm-judge/predefined
# Defining custom scorers: https://docs.databricks.com/aws/en/mlflow3/genai/eval-monitor/custom-scorers
eval_dataset = [
    {
        "inputs": {
            "request": {
                "input": [{"role": "user", "content": "Calculate the 15th Fibonacci number"}]
            }
        },
        "expected_response": "The 15th Fibonacci number is 610.",
    }
]

# Get the invoke function that was registered via @invoke decorator in your agent
invoke_fn = get_invoke_function()
assert invoke_fn is not None, (
    "No function registered with the `@invoke` decorator found."
    "Ensure you have a function decorated with `@invoke()`."
)

# if invoke function is async, then we need to wrap it in a sync function
if asyncio.iscoroutinefunction(invoke_fn):

    def sync_invoke_fn(request: dict) -> ResponsesAgentResponse:
        req = ResponsesAgentRequest(**request)
        return asyncio.run(invoke_fn(req))
else:
    sync_invoke_fn = invoke_fn


def evaluate():
    mlflow.genai.evaluate(
        data=eval_dataset,
        predict_fn=sync_invoke_fn,
        scorers=[RelevanceToQuery(), Safety()],
    )
