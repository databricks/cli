import argparse
import dataclasses
import sys
import re
import copy
from pathlib import Path
from typing import Dict

import yaml
import jinja2

from dbx2dab.compare import (
    recursive_compare,
    recursive_intersection,
    recursive_subtract,
    recursive_merge,
    walk,
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

        # Filter out keys that are not in the base configuration
        filtered_ordered_keys = [k for k in ordered_keys if k in obj]

        return {
            "resources": {
                "jobs": {
                    self.normalized_key(): {k: obj[k] for k in filtered_ordered_keys}
                }
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


class LookupRewriter:
    @dataclasses.dataclass
    class RewriteType:
        variable_name_suffix: str
        object_type: str

    _prefixes = {
        "cluster://": RewriteType(
            variable_name_suffix="cluster_id",
            object_type="cluster",
        ),
        "cluster-policy://": RewriteType(
            variable_name_suffix="cluster_policy_id",
            object_type="cluster_policy",
        ),
        "instance-profile://": None,
        "instance-pool://": RewriteType(
            variable_name_suffix="instance_pool_id",
            object_type="instance_pool",
        ),
        "pipeline://": None,
        "service-principal://": RewriteType(
            variable_name_suffix="service_principal_id",
            object_type="service_principal",
        ),
        "warehouse://": RewriteType(
            variable_name_suffix="warehouse_id",
            object_type="warehouse",
        ),
        "query://": None,
        "dashboard://": None,
        "alert://": None,
    }

    def __init__(self, job: Job) -> None:
        """
        One instance per job.
        We track all references by the env they appear in so we can differentiate between them if needed.
        """
        self.job = job
        self.variables = {}

    def add(self, env: str):
        def cb(path, obj):
            if isinstance(obj, str):
                for prefix in self._prefixes.keys():
                    if obj.startswith(prefix):
                        payload = obj.replace(prefix, "")
                        if prefix in self.variables[env]:
                            raise ValueError(
                                f"Duplicate variable reference for {prefix} in {env}"
                            )
                        self.variables[env][str(path)] = [prefix, payload]
                        break
            return obj

        self.variables[env] = dict()
        walk(self.job.configs[env], cb)

    def confirm_envs_are_idential(self) -> Dict[str, any]:
        # Run a deep equal on the dicts for every env
        keys = list(self.variables.keys())
        first = self.variables[keys[0]]
        for key in keys[1:]:
            diff = recursive_subtract(self.variables[key], first)
            if diff:
                raise ValueError("Variable references differ between environments")
        return first

    def rewrite(self) -> Dict[str, any]:
        """
        Returns variables of the form:

        {
            "etl_cluster_policy_id": {
                "description": "<unknown>",
                "lookup": {
                    "cluster_policy": "some_policy"
                }
            }
        }
        """

        rewrites = dict()
        variables = []

        # Compile a list of variables and how to rewrite the existing instances
        for path, (prefix, payload) in self.confirm_envs_are_idential().items():
            rewrite = self._prefixes[prefix]
            if rewrite is None:
                raise ValueError(f"Unhandled prefix: {prefix}")

            variable_name = (
                f"{self.job.normalized_key()}_{rewrite.variable_name_suffix}"
            )

            # Add rewrite for the path
            rewrites[path] = f"${{var.{variable_name}}}"

            # Add variable for the lookup
            variables.append(
                {
                    "name": variable_name,
                    "lookup_type": rewrite.object_type,
                    "lookup_value": payload,
                }
            )

        # Now rewrite the job configuration
        def cb(path, obj):
            rewrite = rewrites.get(str(path), None)
            if rewrite is not None:
                return rewrite
            return obj

        for env in self.job.configs.keys():
            self.job.configs[env] = walk(copy.deepcopy(self.job.configs[env]), cb)

        return variables


class FileRefRewriter:
    _prefixes = [
        "file://",
        "file:fuse://",
    ]

    def __init__(self, job: Job) -> None:
        self.job = job

    def rewrite(self):
        # Now rewrite the job configuration
        def cb(path, obj):
            if not isinstance(obj, str):
                return obj

            for prefix in self._prefixes:
                if not obj.startswith(prefix):
                    continue

                # DBX anchors paths at the root of the project.
                # DABs anchors paths relative to the location of the configuration file.
                # Below we write configuration to the "./resources" directory.
                # We need to go up one level to get to the root of the project.
                payload = obj.replace(prefix, "")
                return f"../{payload}"

            return obj

        for env in self.job.configs.keys():
            self.job.configs[env] = walk(copy.deepcopy(self.job.configs[env]), cb)

        return


def dedup_variables(variables):
    deduped = dict()
    for v in variables:
        if v not in deduped:
            deduped[v] = None
    return deduped.keys()


def save_databricks_yml(
    base_path: Path, env_variables, var_variables, var_lookup_variables
):
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
                var_lookup_variables=var_lookup_variables,
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
    var_lookup_variables = []

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

    # Locate variable lookups
    for job in jobs.values():
        lr = LookupRewriter(job)
        for env in job.configs:
            lr.add(env)
        var_lookup_variables.extend(lr.rewrite())

    # Rewrite file references
    for job in jobs.values():
        fr = FileRefRewriter(job)
        fr.rewrite()

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
    save_databricks_yml(base_path, env_variables, var_variables, var_lookup_variables)

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
