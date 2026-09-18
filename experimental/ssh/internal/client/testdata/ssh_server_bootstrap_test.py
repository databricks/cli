#!/usr/bin/env python3
"""Test notebook linger helpers without importing Databricks runtime dependencies."""

import ast
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import Mock, call


class LingerTest(unittest.TestCase):
    def test_wait_for_detached_descendants(self):
        source = Path(__file__).resolve().parents[1] / "ssh-server-bootstrap.py"
        module = ast.parse(source.read_text())
        module.body = [
            node
            for node in module.body
            if isinstance(node, ast.FunctionDef) and node.name in {"has_children", "wait_for_detached_descendants"}
        ]
        code = compile(module, str(source), "exec")

        cases = {
            "no children": ([[]], [ChildProcessError()], [], 0),
            "child awaiting adoption": ([[], ["42"], []], [None, ChildProcessError()], [0, 1], 2),
            "exited child awaiting reaping": ([[], []], [object(), ChildProcessError()], [0], 1),
            "adopted child": ([["42"], []], [ChildProcessError()], [0], 1),
            "pending adoption spans multiple polls": (
                [[], [], ["42"], []],
                [None, None, ChildProcessError()],
                [0, 1, 2],
                2,
            ),
            "pending adoption report interval": ([[], [], []], [None, None, ChildProcessError()], [0, 300], 2),
        }
        for name, (survivors, children, timestamps, report_count) in cases.items():
            with self.subTest(name=name):
                mock_os = SimpleNamespace(P_ALL=0, WEXITED=1, WNOHANG=2, WNOWAIT=4, waitid=Mock(side_effect=children))
                mock_time = SimpleNamespace(monotonic=Mock(side_effect=timestamps), sleep=Mock())
                namespace = {
                    "os": mock_os,
                    "time": mock_time,
                    "detached_descendants": Mock(side_effect=survivors),
                    "LINGER_POLL_SECONDS": 1,
                    "LINGER_REPORT_SECONDS": 300,
                    "print": Mock(),
                }
                exec(code, namespace)

                namespace["wait_for_detached_descendants"](123)

                self.assertEqual(namespace["detached_descendants"].call_args_list, [call(123)] * len(survivors))
                self.assertEqual(
                    mock_os.waitid.call_args_list,
                    [call(mock_os.P_ALL, 0, mock_os.WEXITED | mock_os.WNOHANG | mock_os.WNOWAIT)] * len(children),
                )
                self.assertEqual(mock_time.sleep.call_args_list, [call(1)] * (len(survivors) - 1))
                self.assertEqual(mock_time.monotonic.call_count, len(timestamps))
                self.assertEqual(namespace["print"].call_count, report_count + 1)
                self.assertIn("No detached processes left", namespace["print"].call_args[0][0])


if __name__ == "__main__":
    unittest.main()
