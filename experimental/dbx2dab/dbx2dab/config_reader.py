import json
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Any, Dict, Optional

import jinja2
import yaml


class _AbstractConfigReader(ABC):
    def __init__(self, path: Path):
        self._path = path
        self.config = self.get_config()

    def get_config(self) -> any:
        return self._read_file()

    @abstractmethod
    def _read_file(self) -> any:
        """"""


class YamlConfigReader(_AbstractConfigReader):
    def _read_file(self) -> any:
        return yaml.load(self._path.read_text(encoding="utf-8"), yaml.SafeLoader)


class JsonConfigReader(_AbstractConfigReader):
    def _read_file(self) -> any:
        return json.loads(self._path.read_text(encoding="utf-8"))


class Jinja2ConfigReader(_AbstractConfigReader):
    def __init__(
        self,
        path: Path,
        ext: str,
        jinja_vars_file: Optional[Path],
        env_proxy=None,
        var_proxy=None,
    ):
        self._ext = ext
        self._jinja_vars_file = jinja_vars_file
        self._env_proxy = env_proxy
        self._var_proxy = var_proxy
        super().__init__(path)

    @staticmethod
    def _read_vars_file(file_path: Path) -> Dict[str, Any]:
        return yaml.load(file_path.read_text(encoding="utf-8"), yaml.SafeLoader)

    @classmethod
    def _render_content(cls, file_path: Path, globals: Dict[str, Any]) -> str:
        absolute_parent_path = file_path.absolute().parent
        file_name = file_path.name

        env = jinja2.Environment(loader=jinja2.FileSystemLoader(absolute_parent_path))
        template = env.get_template(file_name)

        return template.render(**globals)

    def _read_file(self) -> any:
        rendered = self._render_content(
            self._path,
            {
                "env": self._env_proxy,
                "var": self._var_proxy,
            },
        )

        if self._ext == ".json":
            _content = json.loads(rendered)
            return _content
        elif self._ext in [".yml", ".yaml"]:
            _content = yaml.load(rendered, yaml.SafeLoader)
            return _content
        else:
            raise Exception(f"Unexpected extension for Jinja reader: {self._ext}")


class ConfigReader:
    """
    Entrypoint for reading the raw configurations from files.
    In most cases there is no need to use the lower-level config readers.
    If a new reader is introduced, it shall be used via the :code:`_define_reader` method.
    """

    def __init__(
        self,
        path: Path,
        jinja_vars_file: Optional[Path] = None,
        env_proxy=None,
        var_proxy=None,
    ):
        self._jinja_vars_file = jinja_vars_file
        self._path = path
        self._env_proxy = env_proxy
        self._var_proxy = var_proxy
        self._reader = self._define_reader()

    def _define_reader(self) -> _AbstractConfigReader:
        if False:
            pass
        # if len(self._path.suffixes) > 1:
        #     if self._path.suffixes[0] in [".json", ".yaml", ".yml"] and self._path.suffixes[1] == ".j2":
        #         dbx_echo(
        #             """[bright_magenta bold]You're using a deployment file with .j2 extension.
        #             if you would like to use Jinja directly inside YAML or JSON files without changing the extension,
        #             you can also configure your project to support in-place Jinja by running:
        #             [code]dbx configure --enable-inplace-jinja-support[/code][/bright_magenta bold]"""
        #         )
        #         return Jinja2ConfigReader(self._path, ext=self._path.suffixes[0], jinja_vars_file=self._jinja_vars_file)
        # elif ProjectConfigurationManager().get_jinja_support():
        #     return Jinja2ConfigReader(self._path, ext=self._path.suffixes[0], jinja_vars_file=self._jinja_vars_file)
        else:
            if self._jinja_vars_file:
                return Jinja2ConfigReader(
                    self._path,
                    ext=self._path.suffixes[0],
                    jinja_vars_file=self._jinja_vars_file,
                    env_proxy=self._env_proxy,
                    var_proxy=self._var_proxy,
                )
            if self._path.suffixes[0] == ".json":
                return JsonConfigReader(self._path)
            elif self._path.suffixes[0] in [".yaml", ".yml"]:
                return YamlConfigReader(self._path)

        # no matching reader found, raising an exception
        raise Exception(
            f"Unexpected extension of the deployment file: {self._path}. "
            f"Please check the documentation for supported extensions."
        )

    def get_config(self) -> any:
        return self._reader.config
