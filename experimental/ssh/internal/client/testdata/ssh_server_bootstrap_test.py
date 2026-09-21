#!/usr/bin/env python3
"""Test notebook linger helpers without importing Databricks runtime dependencies."""

import importlib.util
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import Mock, call, patch


class LingerTest(unittest.TestCase):
    def test_wait_for_detached_descendants(self):
        source = Path(__file__).resolve().parents[1] / "ssh-server-bootstrap.py"
        spec = importlib.util.spec_from_file_location("ssh_server_bootstrap", source)
        bootstrap = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(bootstrap)

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
                mock_descendants = Mock(side_effect=survivors)
                with (
                    patch.object(bootstrap, "os", mock_os),
                    patch.object(bootstrap, "time", mock_time),
                    patch.object(bootstrap, "detached_descendants", mock_descendants),
                    patch("builtins.print") as mock_print,
                ):
                    bootstrap.wait_for_detached_descendants(123)

                self.assertEqual(mock_descendants.call_args_list, [call(123)] * len(survivors))
                self.assertEqual(
                    mock_os.waitid.call_args_list,
                    [call(mock_os.P_ALL, 0, mock_os.WEXITED | mock_os.WNOHANG | mock_os.WNOWAIT)] * len(children),
                )
                self.assertEqual(
                    mock_time.sleep.call_args_list, [call(bootstrap.LINGER_POLL_SECONDS)] * (len(survivors) - 1)
                )
                self.assertEqual(mock_time.monotonic.call_count, len(timestamps))
                self.assertEqual(mock_print.call_count, report_count + 1)
                self.assertIn("No detached processes left", mock_print.call_args[0][0])


if __name__ == "__main__":
    unittest.main()
