"""Offline regression check: staged and subsequently deleted secrets must fail.

Only synthetic values are generated, inside a temporary repository. Neither
values nor scanner output are printed. No remote or real credentials are used.
"""
from pathlib import Path
import shutil
import random
import string
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent.parent


def run(args, cwd, expected=0):
    result = subprocess.run(args, cwd=cwd, capture_output=True, text=True)
    if result.returncode != expected:
        raise RuntimeError(f'{args[0]} returned {result.returncode}; expected {expected}. Output withheld to prevent secret disclosure.')


with tempfile.TemporaryDirectory(prefix='nummion-secret-check-') as directory:
    repo = Path(directory)
    run(['git', 'init', '-q'], repo)
    run(['git', 'config', 'user.name', 'Scanner Test'], repo)
    run(['git', 'config', 'user.email', 'scanner@example.invalid'], repo)
    run(['git', 'config', 'core.hooksPath', str(repo / 'no-hooks')], repo)
    shutil.copy(ROOT / '.gitleaks.toml', repo / '.gitleaks.toml')
    (repo / 'README.md').write_text('Synthetic secret-scanner regression fixture.\n')
    run(['git', 'add', '.'], repo)
    scan = ['gitleaks', 'git', '.', '--redact=100', '--no-banner', '--ignore-gitleaks-allow']
    run(scan + ['--staged', '--pre-commit'], repo)
    run(['git', 'commit', '-qm', 'Clean baseline'], repo)
    # Construct a token-shaped, nonfunctional value only at runtime. A test
    # filename proves that blanket *_test.go allowlists cannot bypass the check.
    value = 'gh' + 'p_' + ('Ab3xY7mN2q' * 4)[:36]
    fixture = repo / 'fixture_test.go'
    fixture.write_text('package fixture\nvar token = "' + value + '"\n')
    run(['git', 'add', '.'], repo)
    run(scan + ['--staged', '--pre-commit'], repo, expected=1)
    lexware = repo / 'auth.env'
    synthetic = ''.join(random.Random(42).sample(string.ascii_letters + string.digits, 40))
    lexware.write_text('LEXWARE_API_KEY=' + synthetic + '\n')
    run(['git', 'add', '.'], repo)
    run(scan + ['--enable-rule=lexware-api-key', '--staged', '--pre-commit'], repo, expected=1)
    run(['git', 'commit', '-qm', 'Synthetic fixture'], repo)
    fixture.unlink()
    lexware.unlink()
    run(['git', 'add', '-u'], repo)
    run(['git', 'commit', '-qm', 'Delete synthetic fixture'], repo)
    run(scan + ['--log-opts=--all --full-history'], repo, expected=1)
print('Secret scanner checks passed: clean content accepted; staged test-file and deleted historical secrets rejected.')
