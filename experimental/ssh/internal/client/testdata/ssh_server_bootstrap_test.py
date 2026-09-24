#!/usr/bin/env python3
"""Test notebook process lifecycle helpers without Databricks runtime dependencies."""

import importlib.util
import signal
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import Mock, call, patch


class FakeClock:
    def __init__(self):
        self.value = 0.0

    def monotonic(self):
        return self.value

    def sleep(self, seconds):
        self.value += seconds


def load_bootstrap():
    source = Path(__file__).resolve().parents[1] / "ssh-server-bootstrap.py"
    spec = importlib.util.spec_from_file_location("ssh_server_bootstrap", source)
    bootstrap = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(bootstrap)
    return bootstrap


class LingerTest(unittest.TestCase):
    def test_wait_for_adopted_children(self):
        bootstrap = load_bootstrap()

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
                mock_time = SimpleNamespace(monotonic=Mock(side_effect=timestamps), sleep=Mock())
                enumerate_children = Mock(side_effect=survivors)
                child_presence = [not isinstance(result, ChildProcessError) for result in children]
                has_children = Mock(side_effect=child_presence)
                with (
                    patch.object(bootstrap, "time", mock_time),
                    patch("builtins.print") as mock_print,
                ):
                    bootstrap.wait_for_adopted_children(123, enumerate_children, has_children)

                self.assertEqual(enumerate_children.call_args_list, [call(123)] * len(survivors))
                self.assertEqual(has_children.call_count, len(children))
                self.assertEqual(
                    mock_time.sleep.call_args_list,
                    [call(bootstrap.LINGER_POLL_SECONDS)] * (len(survivors) - 1),
                )
                self.assertEqual(mock_time.monotonic.call_count, len(timestamps))
                self.assertEqual(mock_print.call_count, report_count + 1)
                self.assertIn("No detached processes left", mock_print.call_args[0][0])


class ProcessOwnershipTest(unittest.TestCase):
    def setUp(self):
        self.bootstrap = load_bootstrap()

    def test_adopted_children_include_server_process_group_and_exclude_server(self):
        proc_root = Path(__file__).resolve().parent / "proc"

        children = self.bootstrap.adopted_children(100, proc_root=proc_root, self_pid=100)

        self.assertEqual(children, ["200", "201"])

    def test_keep_mode_preserves_same_group_survivors_without_signaling(self):
        proc_root = Path(__file__).resolve().parent / "proc"
        children = self.bootstrap.adopted_children(100, proc_root=proc_root, self_pid=100)
        enumerate_children = Mock(side_effect=[children, []])
        mock_time = SimpleNamespace(monotonic=Mock(return_value=0), sleep=Mock())

        with (
            patch.object(self.bootstrap.os, "kill") as kill,
            patch.object(self.bootstrap, "time", mock_time),
            patch("builtins.print"),
        ):
            self.bootstrap.wait_for_adopted_children(100, enumerate_children, Mock(return_value=False))

        self.assertEqual(children, ["200", "201"])
        self.assertEqual(enumerate_children.call_args_list, [call(100), call(100)])
        mock_time.sleep.assert_called_once_with(self.bootstrap.LINGER_POLL_SECONDS)
        kill.assert_not_called()

    def test_sweep_signals_each_child_once_before_escalating(self):
        alive = {"200", "201"}
        signals = []
        clock = FakeClock()

        def enumerate_children(server_pid):
            self.assertEqual(server_pid, 100)
            return sorted(alive)

        def signal_child(child_pid, sig):
            signals.append((child_pid, sig))
            if sig == signal.SIGKILL:
                alive.remove(child_pid)

        self.bootstrap.sweep_adopted_children(
            100,
            enumerate_children=enumerate_children,
            signal_child=signal_child,
            monotonic=clock.monotonic,
            sleep=clock.sleep,
        )

        self.assertEqual(
            signals,
            [
                ("200", signal.SIGTERM),
                ("201", signal.SIGTERM),
                ("200", signal.SIGKILL),
                ("201", signal.SIGKILL),
            ],
        )
        self.assertAlmostEqual(clock.value, self.bootstrap.CHILD_SWEEP_TIMEOUT_SECONDS)

    def test_sweep_does_not_kill_child_that_exits_after_sigterm(self):
        alive = {"200"}
        signals = []
        clock = FakeClock()

        def enumerate_children(server_pid):
            return sorted(alive)

        def signal_child(child_pid, sig):
            signals.append((child_pid, sig))
            alive.clear()

        self.bootstrap.sweep_adopted_children(
            100,
            enumerate_children=enumerate_children,
            signal_child=signal_child,
            monotonic=clock.monotonic,
            sleep=clock.sleep,
        )

        self.assertEqual(signals, [("200", signal.SIGTERM)])

    def test_sweep_times_out_if_child_remains(self):
        signals = []
        clock = FakeClock()

        def enumerate_children(server_pid):
            return ["200"]

        def signal_child(child_pid, sig):
            signals.append((child_pid, sig))

        with self.assertRaisesRegex(TimeoutError, "200"):
            self.bootstrap.sweep_adopted_children(
                100,
                enumerate_children=enumerate_children,
                signal_child=signal_child,
                monotonic=clock.monotonic,
                sleep=clock.sleep,
            )
        self.assertEqual(signals, [("200", signal.SIGTERM), ("200", signal.SIGKILL)])
        self.assertAlmostEqual(clock.value, 2 * self.bootstrap.CHILD_SWEEP_TIMEOUT_SECONDS)


if __name__ == "__main__":
    unittest.main()
