import contextlib
import hashlib
import io
import json
import pathlib
import subprocess
import tempfile
import unittest
from unittest import mock

import yaml

import qualify


class QualificationRecordTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = pathlib.Path(self.tmp.name)
        self.run_dir = self.root / "release-gate" / "qualification-scaleway"
        self.bundle = self.run_dir / "bundle"
        self.bundle.mkdir(parents=True)
        tenant = self.root / "references" / "qualification" / "scaleway"
        tenant.mkdir(parents=True)
        (tenant / "tenant.yaml").write_text(yaml.safe_dump({"name_prefix": "qual-vm-"}))
        self.expected = tenant / "expected.yaml"
        self.expected.write_text(yaml.safe_dump({
            "exit_codes": {"live": 1},
            "controls": {
                "a_fail": {"live": {"fail": ["${output.vm}"], "silent": ["qual-vm-ok"]}},
                "b_pass": {"live": "pass"},
                "c_evaluated": {"live": {"status": "evaluated", "reason": "outside tenant"}},
                "d_defect": {"live": {"fail": ["${output.vm}"], "known_defect": "#1"}},
                "e_missing": {"live": {"fail": ["qual-vm-missing"]}},
                "f_silent": {"live": {"silent": ["qual-vm-ok"]}},
            },
        }))
        (self.run_dir / "outputs.json").write_text(json.dumps({"vm": "qual-vm-bad"}))
        self.assessment = {
            "run": {
                "source": "live-api",
                "timestamp": "2026-09-11T00:01:00Z",
                "target": {"provider": "scaleway", "id": "private-tenant-id"},
                "tool": {"version": "0.1.0-dev", "digest": "vcs:abcdef123456+modified"},
            },
            "results": [
                {"control": "a_fail", "status": "fail", "subject": "qual-vm-bad"},
                {"control": "b_pass", "status": "pass", "subject": "scaleway"},
                {"control": "c_evaluated", "status": "pass", "subject": "outside-tenant"},
                {"control": "d_defect", "status": "fail", "subject": "qual-vm-bad"},
            ],
        }
        self.report = {
            "stage": "qualification-scaleway",
            "plan_only": False,
            "verdict": "GO",
            "regression_contract": "GO",
            "destroyed": True,
            "leftovers": {"tagged": [], "delta": {}},
            "started": "2026-09-11T00:00:00Z",
            "duration_seconds": 120,
            "comparison": {"live": {"ok": True, "problems": [], "exit_code": 1,
                                     "results": len(self.assessment["results"]), "known_defects": []}},
            "stages": [{"name": name, "rc": 0} for name in (
                "apply", "scan --live, 5 formats, bundle scellé",
                "bundle : verify --re-derive, refus de l'altération", "destroy",
                "preuve de destruction : listing par famille + delta",
                "comparaison à expected.yaml, et falsification de la porte")],
        }
        self.args = mock.Mock(run_dir=str(self.run_dir), provider=None)
        self.root_patch = mock.patch.object(qualify, "ROOT", self.root)
        self.root_patch.start()
        self.addCleanup(self.root_patch.stop)
        self.verify_patch = mock.patch.object(
            qualify.subprocess, "run", return_value=subprocess.CompletedProcess([], 0))
        self.verify = self.verify_patch.start()
        self.addCleanup(self.verify_patch.stop)
        self.write_files()

    def write_files(self):
        (self.run_dir.parent / "qualification-scaleway.json").write_text(json.dumps(self.report))
        assessment = json.dumps(self.assessment)
        (self.run_dir / "assessment-live.json").write_text(assessment)
        (self.bundle / "assessment.json").write_text(assessment)

    def record(self):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            self.assertEqual(qualify.cmd_record(self.args), 0)
        return json.loads(out.getvalue()), out.getvalue()

    def test_only_explicit_verified_verdicts_are_public(self):
        record, raw = self.record()
        self.assertEqual(record["recorded_at"], "2026-09-11T00:01:00Z")
        self.assertEqual(record["reevaluated_expected_sha256"],
                         hashlib.sha256(self.expected.read_bytes()).hexdigest())
        self.assertEqual({(o["control"], o["verdict"]) for o in record["observations"]},
                         {("a_fail", "fail"), ("b_pass", "pass")})
        self.assertNotIn("private-tenant-id", raw)
        self.assertNotIn("qual-vm-", raw)
        self.verify.assert_called_once()

    def test_incomplete_or_failed_qualification_is_rejected(self):
        for field, value in (("plan_only", True), ("destroyed", False), ("verdict", "NO-GO")):
            with self.subTest(field=field):
                original = self.report[field]
                self.report[field] = value
                self.write_files()
                with self.assertRaisesRegex(ValueError, "qualification live GO"):
                    self.record()
                self.report[field] = original

    def test_leftovers_and_missing_or_duplicate_stages_are_rejected(self):
        self.report["leftovers"]["delta"] = {"untracked": ["qual-vm-orphan"]}
        self.write_files()
        with self.assertRaisesRegex(ValueError, "destruction"):
            self.record()
        self.report["leftovers"]["delta"] = {}
        self.report["stages"].pop()
        self.write_files()
        with self.assertRaisesRegex(ValueError, "étape obligatoire"):
            self.record()
        self.report["stages"].append(self.report["stages"][0])
        self.write_files()
        with self.assertRaisesRegex(ValueError, "dupliquées"):
            self.record()

    def test_invalid_bundle_and_mismatched_metadata_are_rejected(self):
        self.verify.return_value = subprocess.CompletedProcess([], 1)
        with self.assertRaisesRegex(ValueError, "bundle incohérent"):
            self.record()
        self.verify.return_value = subprocess.CompletedProcess([], 0)
        changed = json.loads((self.run_dir / "assessment-live.json").read_text())
        changed["run"]["timestamp"] = "2026-09-11T00:01:01Z"
        (self.run_dir / "assessment-live.json").write_text(json.dumps(changed))
        with self.assertRaisesRegex(ValueError, "métadonnées"):
            self.record()

    def test_provider_and_live_source_are_checked(self):
        for field, value in (("source", "terraform"), ("provider", "exoscale")):
            with self.subTest(field=field):
                original = json.loads(json.dumps(self.assessment))
                if field == "source":
                    self.assessment["run"]["source"] = value
                else:
                    self.assessment["run"]["target"]["provider"] = value
                self.write_files()
                with self.assertRaisesRegex(ValueError, "collecte live"):
                    self.record()
                self.assessment = original


if __name__ == "__main__":
    unittest.main()
