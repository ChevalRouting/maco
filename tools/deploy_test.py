import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import tempfile
import unittest

TOOLS = Path(__file__).resolve().parent
MOCK = '''#!/usr/bin/env python3
import json, os, pathlib, subprocess, sys
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ['DEPLOY_TEST_LOG'], 'a') as log:
    log.write(json.dumps([name, *args]) + '\\n')
if os.environ.get('DEPLOY_TEST_FAIL') == name:
    sys.exit(1)
if name == 'ssh' and args[-1] == 'mktemp -d /tmp/maco-deploy.XXXXXX':
    print('/tmp/maco-deploy.test123')
if name == 'sudo' and args != ['-v']:
    sys.exit(subprocess.call(args))
if name == 'maco' and args == ['service', 'uninstall'] and os.environ.get('DEPLOY_TEST_FAIL') == 'uninstall':
    sys.exit(1)
'''


class DeployTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        self.log = self.root / 'calls.jsonl'
        self.env = dict(os.environ, PATH=f'{self.bin}:{os.environ["PATH"]}',
                        DEPLOY_TEST_LOG=str(self.log))
        for name in ('ssh', 'scp', 'sudo', 'codesign'):
            self.write_mock(self.bin / name)
        self.write_mock(self.root / 'maco')
        shutil.copyfile(TOOLS / 'deploy-install.sh', self.root / 'install.sh')

    def write_mock(self, path):
        path.write_text(MOCK)
        path.chmod(0o755)

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def run_script(self, script, *args):
        return subprocess.run(['bash', str(script), *args], env=self.env,
                              capture_output=True, text=True)

    def test_upload_staging_and_argument_quoting(self):
        result = self.run_script(TOOLS / 'deploy.sh', 'admin@mac',
                                 str(self.root / 'maco'), '--data-dir',
                                 '/Library/Application Support/maco')
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self.calls()
        self.assertEqual([c[0] for c in calls], ['ssh', 'scp', 'scp', 'ssh', 'ssh'])
        self.assertEqual(calls[1][-1], 'admin@mac:/tmp/maco-deploy.test123/maco')
        self.assertIn('-t', calls[3])
        self.assertEqual(shlex.split(calls[3][-1]), [
            '/bin/bash', '/tmp/maco-deploy.test123/install.sh',
            '--data-dir', '/Library/Application Support/maco'])
        self.assertIn('rm -rf', calls[-1][-1])

    def test_copy_failure_does_not_install(self):
        self.env['DEPLOY_TEST_FAIL'] = 'scp'
        result = self.run_script(TOOLS / 'deploy.sh', 'admin@mac', str(self.root / 'maco'))
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any('-t' in call for call in self.calls()))

    def test_sign_stop_install_order(self):
        result = self.run_script(self.root / 'install.sh', '--addr', ':8443')
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self.calls()
        self.assertEqual(calls[0], ['codesign', '--sign', '-', '--force',
                                   '--preserve-metadata=entitlements,requirements,flags,runtime', 'maco'])
        self.assertEqual(calls[1], ['codesign', '--verify', '--strict', 'maco'])
        self.assertEqual([c for c in calls if c[0] == 'maco'], [
            ['maco', 'service', 'uninstall'], ['maco', 'install', '--addr', ':8443']])

    def test_uninstall_failure_does_not_install(self):
        self.env['DEPLOY_TEST_FAIL'] = 'uninstall'
        result = self.run_script(self.root / 'install.sh')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(c[:2] == ['maco', 'install'] for c in self.calls()))

    def test_sign_failure_does_not_stop_service(self):
        self.env['DEPLOY_TEST_FAIL'] = 'codesign'
        result = self.run_script(self.root / 'install.sh')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(self.calls()), 1)


if __name__ == '__main__':
    unittest.main()
