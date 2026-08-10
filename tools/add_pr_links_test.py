#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Unit tests for add_pr_links. Run with ``uv run tools/add_pr_links_test.py``."""

import pathlib
import tempfile
import unittest

from add_pr_links import annotate_text, append_reference, entry_ranges, process_file

PR = 6177
# The link add_pr_links writes and update_github_links later expands.
REF = f"(#{PR})"


class EntryRanges(unittest.TestCase):
    def test_bullets_split_into_one_range_each(self):
        self.assertEqual(entry_ranges(["* first", "* second"]), [(0, 1), (1, 2)])

    def test_dash_marker_also_starts_an_entry(self):
        self.assertEqual(entry_ranges(["- first", "- second"]), [(0, 1), (1, 2)])

    def test_no_marker_is_a_single_entry(self):
        self.assertEqual(entry_ranges(["one entry", "wrapped over two lines"]), [(0, 2)])

    def test_continuation_lines_belong_to_their_bullet(self):
        self.assertEqual(entry_ranges(["* first", "  continued", "* second"]), [(0, 2), (2, 3)])


class AppendReference(unittest.TestCase):
    def test_appended_after_a_trailing_period(self):
        self.assertEqual(append_reference("Added a command.", PR), f"Added a command. {REF}")

    def test_appended_when_there_is_no_trailing_period(self):
        self.assertEqual(append_reference("* No trailing period", PR), f"* No trailing period {REF}")

    def test_trailing_whitespace_is_stripped_first(self):
        self.assertEqual(append_reference("entry   ", PR), f"entry {REF}")


class AnnotateText(unittest.TestCase):
    def test_reference_goes_at_the_very_end(self):
        self.assertEqual(annotate_text("Added a command.\n", PR), f"Added a command. {REF}\n")

    def test_entries_are_annotated_independently(self):
        got = annotate_text("* one\n* two\n", PR)
        self.assertEqual(got, f"* one {REF}\n* two {REF}\n")

    def test_entry_ending_in_a_raw_reference_is_left_untouched(self):
        text = "Added a command (#99).\n"
        self.assertEqual(annotate_text(text, PR), text)

    def test_entry_ending_in_an_expanded_link_is_left_untouched(self):
        text = "* two ([#99](https://github.com/databricks/cli/pull/99))\n"
        self.assertEqual(annotate_text(text, PR), text)

    def test_reference_in_the_body_does_not_block_appending(self):
        # A prior PR/issue cited mid-entry must not be mistaken for the entry's
        # own reference; the PR still goes at the end.
        got = annotate_text("Fixes #6030: a bug.\n", PR)
        self.assertEqual(got, f"Fixes #6030: a bug. {REF}\n")

    def test_issue_link_at_the_end_does_not_block_appending(self):
        # END_REF_RE requires a /pull/ link, so an issue link doesn't count.
        text = "See [#6030](https://github.com/databricks/cli/issues/6030).\n"
        got = annotate_text(text, PR)
        self.assertEqual(got, f"See [#6030](https://github.com/databricks/cli/issues/6030). {REF}\n")

    def test_only_the_unreferenced_entry_in_a_mix_is_annotated(self):
        text = "* one\n* two ([#99](https://github.com/databricks/cli/pull/99))\n"
        got = annotate_text(text, PR)
        self.assertEqual(got, f"* one {REF}\n* two ([#99](https://github.com/databricks/cli/pull/99))\n")

    def test_wrapped_entry_gets_the_reference_on_its_last_line(self):
        got = annotate_text("A long entry that wraps\nover two lines.\n", PR)
        self.assertEqual(got, f"A long entry that wraps\nover two lines. {REF}\n")

    def test_leading_blank_lines_are_skipped(self):
        got = annotate_text("\n\nLeading blank lines.\n", PR)
        self.assertEqual(got, f"\n\nLeading blank lines. {REF}\n")

    def test_blank_entry_is_not_annotated(self):
        self.assertEqual(annotate_text("", PR), "")
        self.assertEqual(annotate_text("\n", PR), "\n")

    def test_reference_already_present_makes_a_rerun_a_noop(self):
        # This is what keeps the workflow's own push from retriggering itself.
        once = annotate_text("Added a command.\n", PR)
        self.assertEqual(annotate_text(once, PR), once)


class ProcessFile(unittest.TestCase):
    def _write(self, text):
        path = pathlib.Path(tempfile.mkdtemp()) / "fragment.md"
        path.write_text(text, encoding="utf-8")
        return path

    def test_returns_true_and_writes_when_a_reference_is_added(self):
        path = self._write("Added a command.\n")
        self.assertTrue(process_file(path, PR))
        self.assertEqual(path.read_text(encoding="utf-8"), f"Added a command. {REF}\n")

    def test_returns_false_and_leaves_the_file_when_already_referenced(self):
        path = self._write("Added a command (#6064).\n")
        self.assertFalse(process_file(path, PR))
        self.assertEqual(path.read_text(encoding="utf-8"), "Added a command (#6064).\n")


if __name__ == "__main__":
    unittest.main()
