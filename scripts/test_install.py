"""Offline installer tests: no downloads, releases, or real user directories."""
import hashlib
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest

INSTALLER = Path(__file__).with_name('install.sh').resolve()


class InstallerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.fixtures = self.root / 'fixtures'
        self.fixtures.mkdir()
        self.commands = self.root / 'commands'
        self.commands.mkdir()
        self.dest = self.root / 'bin with spaces'
        self.dest.mkdir()
        self.old = self.dest / 'num'
        self.old.write_text('old installation')
        self.env = dict(os.environ, PATH=f'{self.commands}:/usr/bin:/bin:/usr/sbin',
                        NUMMION_VERSION='1.2.3', NUMMION_BIN_DIR=str(self.dest),
                        NUMMION_TEST_FIXTURES=str(self.fixtures))
        self.command('uname', '#!/bin/sh\ncase "$1" in -s) echo Linux;; -m) echo x86_64;; esac\n')
        self.command('curl', '''#!/bin/sh
out=''
while [ "$#" -gt 0 ]; do
    case "$1" in
      -o) out=$2; shift 2;;
      -w|--proto|--tlsv1.2|--retry)
        if [ "$1" = --tlsv1.2 ]; then shift; else shift 2; fi;;
      https:*) url=$1; shift;;
      *) shift;;
    esac
done
if [ "${NUMMION_TEST_NETWORK_ERROR:-}" = 1 ]; then exit 22; fi
if [ "$out" = /dev/null ]; then
    printf https://github.com/JangoCG/nummion/releases/tag/v1.2.3
else
    cp "$NUMMION_TEST_FIXTURES/${url##*/}" "$out"
fi
''')
        self.archive = self.fixtures / 'nummion_1.2.3_linux_amd64.tar.gz'
        self.make_archive()

    def command(self, name, script):
        p = self.commands / name
        p.write_text(script)
        p.chmod(0o755)

    def make_archive(self, version='1.2.3', member='num', symlink=False):
        with tarfile.open(self.archive, 'w:gz') as tar:
            entry = tarfile.TarInfo(member)
            entry.mode = 0o755
            if symlink:
                entry.type = tarfile.SYMTYPE
                entry.linkname = '/usr/bin/true'
                tar.addfile(entry)
            else:
                content = f'#!/bin/sh\nprintf "num {version}\\n"\n'.encode()
                entry.size = len(content)
                tar.addfile(entry, io.BytesIO(content))
        checksum = hashlib.sha256(self.archive.read_bytes()).hexdigest()
        (self.fixtures / 'checksums.txt').write_text(f'{checksum}  {self.archive.name}\n')

    def run_installer(self):
        return subprocess.run(['bash', str(INSTALLER)], env=self.env,
                              capture_output=True, text=True)

    def assert_failure_preserves_install(self):
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.old.read_text(), 'old installation')
        self.assertEqual(list(self.dest.glob('.num-install.*')), [])
        return result

    def test_install_and_upgrade_with_spaces_in_destination(self):
        result = self.run_installer()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(subprocess.check_output([str(self.old), '--version'], text=True), 'num 1.2.3\n')

    def test_latest_release(self):
        del self.env['NUMMION_VERSION']
        self.assertEqual(self.run_installer().returncode, 0)

    def test_corrupt_archive(self):
        self.archive.write_bytes(b'corrupted')
        self.assertIn('SHA-256 mismatch', self.assert_failure_preserves_install().stderr)

    def test_missing_checksum(self):
        (self.fixtures / 'checksums.txt').write_text('')
        self.assert_failure_preserves_install()

    def test_duplicate_checksum(self):
        p = self.fixtures / 'checksums.txt'
        p.write_text(p.read_text() * 2)
        self.assert_failure_preserves_install()

    def test_network_error(self):
        self.env['NUMMION_TEST_NETWORK_ERROR'] = '1'
        self.assert_failure_preserves_install()

    def test_invalid_version(self):
        self.env['NUMMION_VERSION'] = '../../bad'
        self.assert_failure_preserves_install()

    def test_wrong_binary_version(self):
        self.make_archive(version='1.2.2')
        self.assert_failure_preserves_install()

    def test_missing_executable(self):
        self.make_archive(member='another-command')
        self.assert_failure_preserves_install()

    def test_symlink_executable(self):
        self.make_archive(symlink=True)
        self.assert_failure_preserves_install()

    def test_signature_failure(self):
        (self.fixtures / 'checksums.txt.bundle').write_text('{}')
        self.command('cosign', '#!/bin/sh\nexit 1\n')
        self.assertIn('Signature verification failed', self.assert_failure_preserves_install().stderr)


if __name__ == '__main__':
    unittest.main()
