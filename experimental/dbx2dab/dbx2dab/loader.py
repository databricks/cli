from pathlib import Path
from typing import Dict, List, Tuple
import yaml

from dbx2dab.config_reader import ConfigReader
from dbx2dab.compare import tuple_walk


class VerboseSafeDumper(yaml.SafeDumper):
    def ignore_aliases(self, data):
        return True


class EnvProxy:
    def __init__(self):
        self.items = []
        pass

    def variable_name(self, name: str):
        return f"ENV_{name}".upper()

    def variables(self):
        return [self.variable_name(item) for item in self.items]

    def interpolation_for_item(self, name: str):
        return f"${{var.{self.variable_name(name)}}}"

    def __getitem__(self, item):
        self.items.append(item)
        return self.interpolation_for_item(item)


class VarProxy:
    def __init__(self, passthrough: Dict[str, any]):
        self.items = []
        self.passthrough = passthrough
        pass

    def variable_name(self, name: str):
        return name.upper()

    def variables(self):
        return [self.variable_name(item) for item in self.items]

    def interpolation_for_item(self, name: str):
        return f"${{var.{self.variable_name(name)}}}"

    def __getitem__(self, item):
        if item in self.passthrough:
            return self.passthrough[item]

        self.items.append(item)
        return self.interpolation_for_item(item)


class Loader:
    def __init__(self, path: Path):
        self._path = path

    def detect_environments(self) -> List[str]:
        """
        Expect the following directory structure:
        - conf/
            - dev/
                - config.yaml
            - stg/
                - config.yaml
            - prod/
                - config.yaml

        This function will return ["dev", "stg", "prod"]
        """
        return [
            item.name for item in self._path.joinpath("conf").iterdir() if item.is_dir()
        ]

    def _get_config(self, environment: str, env_proxy, var_proxy) -> any:
        deployment_file = self._path.joinpath("conf/deployment.yaml")
        variables_file = self._path.joinpath(f"conf/{environment}/config.yaml")

        config_reader = ConfigReader(
            deployment_file,
            variables_file,
            env_proxy=env_proxy,
            var_proxy=var_proxy,
        )

        config = config_reader.get_config()
        return config["environments"][environment]

    def load_variables_for_environment(self, environment: str) -> Dict[str, any]:
        file = self._path.joinpath(f"conf/{environment}/config.yaml")
        obj: Dict[str, any] = yaml.load(
            file.read_text(encoding="utf-8"), yaml.SafeLoader
        )
        return obj

    def compute_variable_allowlist(self, environment: str) -> List[str]:
        variables_ref = self.load_variables_for_environment(environment)
        env_proxy = EnvProxy()

        # Interpolate all variables.
        complete_config = self._get_config(
            environment, EnvProxy(), VarProxy(passthrough=variables_ref)
        )

        # We will try to find the impact of each variable on the configuration.
        # All variables that can be replaced by a bundle variable will be added to the allowlist.
        variable_allowlist = []

        # Now look for the impact of each variable
        for key in variables_ref.keys():
            dup = dict(variables_ref)
            dup.pop(key)

            # If we cannot use a bundle variable for this key, continue.
            try:
                var_proxy = VarProxy(passthrough=dup)
                config = self._get_config(environment, env_proxy, var_proxy)
            except Exception:
                continue

            # If we can use a bundle variable for this key, look at the difference.
            # We care that every use of the variable is replaced by the bundle variable.
            # If it is replaced by something else, there is logic we cannot capture.
            correctly_replaced = True

            def update_callback(path, old_value, new_value):
                nonlocal correctly_replaced
                if (
                    isinstance(new_value, str)
                    and var_proxy.interpolation_for_item(key) in new_value
                ):
                    return
                print(
                    f"Detected incompatible replacement for {key}: {path}: {old_value} -> {new_value}"
                )
                correctly_replaced = False
                return

            tuple_walk(complete_config, config, update_callback=update_callback)

            # If we found an incompatible replacement, bail.
            if not correctly_replaced:
                continue

            variable_allowlist.append(key)

        return variable_allowlist

    def load_for_environment(self, environment: str) -> Tuple[any, EnvProxy, VarProxy]:
        variables_ref = self.load_variables_for_environment(environment)
        variable_allowlist = self.compute_variable_allowlist(environment)

        # Remove all variables that are not in the allowlist.
        for key in variable_allowlist:
            variables_ref.pop(key)

        env_proxy = EnvProxy()
        var_proxy = VarProxy(passthrough=variables_ref)
        config = self._get_config(environment, env_proxy, var_proxy)
        return config, env_proxy, var_proxy


if __name__ == "__main__":
    """
    Below is an example of how to use the Loader class.

    Use this for testing and debugging purposes only.
    """

    loader = Loader(
        Path(__file__).parent / "../../databricks-etl-notebook-template-main"
    )

    for env in ["dev", "stg", "prod"]:
        loader.load_for_environment(env)
