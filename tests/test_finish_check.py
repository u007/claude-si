import os
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path

sys.dont_write_bytecode = True
ROOT = Path(__file__).resolve().parent.parent
SCRIPT = ROOT / "code-quality" / "scripts" / "finish-check.sh"


class FinishCheck(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.project = Path(self.tmp.name)

    def run_check(self, *args, env=None):
        return subprocess.run(
            ["sh", str(SCRIPT), *args], capture_output=True, text=True, check=False, env=env
        )

    def test_clean_project(self):
        (self.project / "main.py").write_text("print('ok')\n")
        result = self.run_check(str(self.project))
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("clean", result.stdout)

    def test_placeholder_marker_reported_without_editing_file(self):
        notes = self.project / "notes.py"
        notes.write_text("# TODO: fix this\n")
        before = notes.read_text()
        result = self.run_check(str(self.project))
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertIn("notes.py:1", result.stdout)
        self.assertEqual(notes.read_text(), before)

    def test_tracked_todo_file_is_skipped(self):
        (self.project / "TODO.md").write_text("- [ ] TODO: tracked deferral\n")
        result = self.run_check(str(self.project))
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_markdown_prose_is_not_scanned(self):
        (self.project / "notes.md").write_text("TODO: prose mention, not a code placeholder\n")
        result = self.run_check(str(self.project))
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_world_readable_env_file_reported(self):
        env = self.project / ".env"
        env.write_text("KEY=value\n")
        env.chmod(0o644)
        result = self.run_check(str(self.project))
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertIn(".env", result.stdout)

    def test_secret_scan_ignores_templates_and_source_names(self):
        for name in (".env", ".env.example", "token.ts", "token.go", "credentials.ts"):
            path = self.project / name
            path.write_text("KEY=value\n")
            path.chmod(0o644)
        result = self.run_check(str(self.project))
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        section = result.stdout.split("== World-readable secret or database files")[1]
        section = section.split("==")[0]
        self.assertIn(".env\n", section)
        self.assertNotIn(".env.example", section)
        self.assertNotIn("token.ts", section)
        self.assertNotIn("token.go", section)
        self.assertNotIn("credentials.ts", section)

    def test_process_with_cwd_in_project_is_reported(self):
        proc = subprocess.Popen(["sleep", "10"], cwd=str(self.project))

        def cleanup():
            proc.terminate()
            proc.wait()

        self.addCleanup(cleanup)
        time.sleep(0.2)
        result = self.run_check(str(self.project))
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertIn(str(proc.pid), result.stdout)

    @unittest.skipIf(os.path.isdir("/proc"), "the /proc branch is always available")
    def test_unverified_when_process_scan_unavailable(self):
        (self.project / "main.py").write_text("print('ok')\n")
        result = self.run_check(str(self.project), env={"PATH": "/usr/bin:/bin"})
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertIn("could not verify", result.stdout)
        self.assertNotIn("finish-check: clean", result.stdout)

    @unittest.skipIf(os.geteuid() == 0, "root ignores permission bits")
    def test_unreadable_directory_warns_and_fails(self):
        locked = self.project / "locked"
        locked.mkdir()
        (locked / "notes.py").write_text("# TODO: hidden\n")
        locked.chmod(0o000)
        self.addCleanup(locked.chmod, 0o755)
        result = self.run_check(str(self.project))
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertIn("scan incomplete", result.stdout)

    def test_invalid_path_exits_two(self):
        result = self.run_check(str(self.project / "missing"))
        self.assertEqual(result.returncode, 2)
        self.assertIn("not a directory", result.stderr)

    def test_skill_wires_script_path(self):
        skill = ROOT / "code-quality" / "skills" / "finish-check" / "SKILL.md"
        body = skill.read_text()
        self.assertIn("${CLAUDE_PLUGIN_ROOT}/scripts/finish-check.sh", body)
        self.assertFalse(
            (ROOT / "code-quality" / "commands" / "finish-check.md").exists()
        )


if __name__ == "__main__":
    unittest.main()
