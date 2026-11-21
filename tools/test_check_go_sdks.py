import os
import shutil
import subprocess
import tempfile
import unittest

from check_go_sdks import SDK_GO_PREFIX, check_sdk_go, find_release


class SDKGoReleaseTest(unittest.TestCase):
    def setUp(self):
        self.repo = tempfile.mkdtemp()
        self.git("init", "--quiet")
        # A release from before the marker existed is tagged but has no marker.
        self.git("commit", "--quiet", "--allow-empty", "-m", "Release sdk-go")
        self.git("tag", "core/v0.0.1")
        self.release("a" * 40, {"auth": "v0.1.0", "core": "v0.1.0"}, ["auth/v0.1.0", "core/v0.1.0"])
        # A merged release that was never tagged isn't published.
        self.release("b" * 40, {"auth": "v0.2.0"}, [])
        # Tagging stopped after auth.
        self.release("c" * 40, {"auth": "v0.3.0", "core": "v0.2.0"}, ["auth/v0.3.0"])

    def tearDown(self):
        shutil.rmtree(self.repo)

    def git(self, *args):
        env = {
            **os.environ,
            "GIT_AUTHOR_NAME": "t",
            "GIT_AUTHOR_EMAIL": "t@t",
            "GIT_COMMITTER_NAME": "t",
            "GIT_COMMITTER_EMAIL": "t@t",
        }
        return subprocess.run(["git", *args], cwd=self.repo, check=True, capture_output=True, text=True, env=env).stdout

    def write(self, path, content):
        os.makedirs(os.path.dirname(os.path.join(self.repo, path)), exist_ok=True)
        with open(os.path.join(self.repo, path), "w") as f:
            f.write(content)
        self.git("add", path)

    def release(self, universe_sha, versions, tags):
        self.write(".codegen/_last_sha", universe_sha + "\n")
        for package, version in versions.items():
            self.write(f"{package}/.package.json", f'{{"package": "{package}"}}')
            self.write(f"{package}/CHANGELOG.md", f"# Changelog\n\n## Release {version} (2026-11-18)\n\n- A change.\n")
        self.git("commit", "--quiet", "-m", "Release sdk-go")
        for tag in tags:
            self.git("tag", tag)

    def test_find_release(self):
        head = self.git("rev-parse", "HEAD").strip()
        self.assertEqual(find_release(self.repo, "c" * 40), head)
        self.assertIsNotNone(find_release(self.repo, "a" * 40))
        self.assertIsNone(find_release(self.repo, "b" * 40))
        self.assertIsNone(find_release(self.repo, "d" * 40))

    def test_check_sdk_go(self):
        auth, core = SDK_GO_PREFIX + "auth", SDK_GO_PREFIX + "core"
        classic = "github.com/databricks/databricks-sdk-go"
        for name, climod_sha, requires, want_errors in [
            ("versions match", "a" * 40, {auth: "v0.1.0", core: "v0.1.0", classic: "v1.0.0"}, []),
            ("version differs", "a" * 40, {auth: "v0.2.0"}, ["has v0.1.0"]),
            ("module not published", "c" * 40, {auth: "v0.3.0", core: "v0.2.0"}, ["hasn't published " + core]),
            ("no release", "b" * 40, {auth: "v0.2.0"}, ["no sdk-go release was cut"]),
        ]:
            with self.subTest(name):
                errors = check_sdk_go(self.repo, climod_sha, requires)
                self.assertEqual(len(errors), len(want_errors), errors)
                for error, want in zip(errors, want_errors, strict=True):
                    self.assertIn(want, error)


if __name__ == "__main__":
    unittest.main()
