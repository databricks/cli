import argparse
import sys
import re
from pathlib import Path
from typing import Dict

import yaml
import jinja2

from dbx2dab.compare import (
    recursive_compare,
    recursive_intersection,
    recursive_subtract,
    recursive_merge,
)

from dbx2dab.loader import Loader


class VerboseSafeDumper(yaml.SafeDumper):
    """
    A YAML dumper that does not use aliases.
    """

    def ignore_aliases(self, data):
        return True


class Job:
    def __init__(self, name: str) -> None:
        self.name = name
        self.configs = dict()

    def normalized_key(self) -> str:
        name = str(self.name)
        # Remove ${foo.bar} with a regex
        name = re.sub(r"\${.*?}", "", name)
        # Remove leading and trailing underscores
        name = re.sub(r"^_+|_+$", "", name)
        return name

    def register_configuration(self, environment: str, config: Dict[str, any]) -> None:
        self.configs[environment] = config

    def all_equal(self) -> bool:
        keys = list(self.configs.keys())
        if len(keys) == 1:
            return True

        for i in range(1, len(keys)):
            if not recursive_compare(self.configs[keys[i - 1]], self.configs[keys[i]]):
                return False

        return True

    def compute_base(self) -> Dict[str, any]:
        keys = list(self.configs.keys())
        out = self.configs[keys[0]]
        for key in keys[1:]:
            out = recursive_intersection(out, self.configs[key])
        return out

    def compute_resource_definition(self) -> Dict[str, any]:
        ordered_keys = [
            "name",
            "tags",
            "schedule",
            "email_notifications",
            "git_source",
            "permissions",
            "tasks",
            "job_clusters",
        ]

        obj = self.compute_base()

        # DBX uses "access_control_list" per job.
        # In DABs we use "permissions" for the whole job.
        obj["permissions"] = [dict(e) for e in obj["access_control_list"]]
        for permission in obj["permissions"]:
            permission["level"] = permission["permission_level"]
            permission.pop("permission_level")

        return {
            "resources": {
                "jobs": {self.normalized_key(): {k: obj[k] for k in ordered_keys}}
            }
        }

    def compute_override_for_environment(self, environment: str) -> Dict[str, any]:
        base = self.compute_base()
        if environment not in self.configs:
            return {}

        config = self.configs[environment]
        override = recursive_subtract(config, base)

        # If the configuration is the same as the base, we don't need to override.
        if not override:
            return {}

        return {
            "targets": {
                environment: {
                    "resources": {
                        "jobs": {
                            self.normalized_key(): recursive_subtract(config, base)
                        }
                    }
                }
            }
        }


def dedup_variables(variables):
    deduped = dict()
    for v in variables:
        if v not in deduped:
            deduped[v] = None
    return deduped.keys()


def save_databricks_yml(base_path: Path, env_variables, var_variables):
    env = jinja2.Environment(
        loader=jinja2.FileSystemLoader(Path(__file__).parent.joinpath("templates"))
    )
    template = env.get_template("databricks.yml.j2")

    base_name = base_path.name
    dst = base_path.joinpath("databricks.yml")
    print("Writing: ", dst)
    with open(dst, "w") as f:
        f.write(
            template.render(
                bundle_name=base_name,
                env_variables=env_variables,
                var_variables=var_variables,
            )
        )


def compute_variables_for_environment(
    environment: str, variables: Dict[str, any]
) -> Dict[str, any]:
    return {
        "targets": {
            environment: {
                "variables": variables,
            }
        }
    }


def main():
    parser = argparse.ArgumentParser(description="Generate Databricks configurations")
    parser.add_argument("dir", help="Path to the DBX project")
    parser.add_argument(
        "-v", "--verbose", action="store_true", help="Print verbose output"
    )
    args = parser.parse_args()
    verbose: bool = args.verbose

    base_path = Path(args.dir)
    loader = Loader(base_path)
    envs = loader.detect_environments()
    print("Detected environments:", envs)

    env_variables = []
    var_variables = []

    jobs: Dict[str, Job] = dict()
    for env in envs:
        config, env_proxy, var_proxy = loader.load_for_environment(env)
        env_variables.extend(env_proxy.variables())
        var_variables.extend(var_proxy.variables())
        for workflow in config["workflows"]:
            name = workflow["name"]
            if name not in jobs:
                jobs[name] = Job(name)

            jobs[name].register_configuration(env, workflow)

    for job in jobs.values():
        base_job = job.compute_base()

        if verbose:
            print("Job:", job.name)

        # Write job configuration to "./resources" directory
        resource_path = base_path.joinpath("resources")
        resource_path.mkdir(exist_ok=True)

        dst = resource_path.joinpath(f"{job.normalized_key()}.yml")
        print("Writing: ", dst)
        with open(dst, "w") as f:
            yaml.dump(
                job.compute_resource_definition(),
                f,
                Dumper=VerboseSafeDumper,
                sort_keys=False,
            )

        for environment, config in job.configs.items():
            diff = recursive_subtract(config, base_job)

            if verbose:
                yaml.dump(
                    diff,
                    sys.stdout,
                    indent=2,
                    Dumper=VerboseSafeDumper,
                    sort_keys=False,
                )

    # Write variable definitions
    env_variables = dedup_variables(env_variables)
    var_variables = dedup_variables(var_variables)
    save_databricks_yml(base_path, env_variables, var_variables)

    # Write resource overrides
    for env in envs:
        out = {}
        for job in jobs.values():
            out = recursive_merge(out, job.compute_override_for_environment(env))

        if out:
            dst = base_path.joinpath(f"conf/{env}/overrides.yml")
            print("Writing: ", dst)
            with open(dst, "w") as f:
                yaml.dump(
                    out,
                    f,
                    Dumper=VerboseSafeDumper,
                    sort_keys=False,
                )

    # Write variable values
    for env in envs:
        variables = loader.load_variables_for_environment(env)

        # Include only variables listed in "var_variables"
        variables = {k: v for k, v in variables.items() if k in var_variables}

        out = compute_variables_for_environment(env, variables)
        if out:
            dst = base_path.joinpath(f"conf/{env}/variables.yml")
            print("Writing: ", dst)
            with open(dst, "w") as f:
                yaml.dump(
                    out,
                    f,
                    Dumper=VerboseSafeDumper,
                    sort_keys=False,
                )


if __name__ == "__main__":
    main()
