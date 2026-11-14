"""Exercise a disposable PocketBase instance, migrations and ownership rules."""
import json
import os
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

BASE = 'http://127.0.0.1:18090'

def request(path, method='GET', body=None, token=None, expected=200):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = token
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=5) as res:
            status, payload = res.status, res.read()
    except urllib.error.HTTPError as err:
        status, payload = err.code, err.read()
    assert status == expected, (path, status, expected, payload.decode())
    return json.loads(payload) if payload and payload.startswith(b'{') else payload

with tempfile.TemporaryDirectory(prefix='haphazard-pb-') as data:
    proc = subprocess.Popen(['/tmp/haphazard-server', 'serve', '--http=127.0.0.1:18090', '--dir=' + data], stdout=subprocess.DEVNULL, stderr=subprocess.STDOUT)
    try:
        for attempt in range(100):
            try:
                request('/api/health')
                break
            except (OSError, urllib.error.URLError):
                if proc.poll() is not None:
                    raise RuntimeError('PocketBase exited during startup')
                time.sleep(.1)
        else:
            raise RuntimeError('PocketBase did not start')
        assert b'Haphazard Martians' in request('/')
        assert len(request('/game.wasm')) > 1_000_000
        users = []
        for name in ('one', 'two'):
            email = name + '@example.test'
            password = 'local-test-password-123'
            user = request('/api/collections/users/records', 'POST', {'email': email, 'password': password, 'passwordConfirm': password}, expected=200)
            auth = request('/api/collections/users/auth-with-password', 'POST', {'identity': email, 'password': password})
            users.append((user['id'], auth['token']))
        uid, token = users[0]
        uid2, token2 = users[1]
        run = {'user': uid, 'level': 1, 'won': True, 'stars': 2, 'score': 700}
        request('/api/collections/hm_runs/records', 'POST', run, expected=400)
        request('/api/collections/hm_runs/records', 'POST', run, token=token2, expected=400)
        created = request('/api/collections/hm_runs/records', 'POST', run, token=token)
        assert created['user'] == uid
        assert any(row['id'] == created['id'] for row in request('/api/collections/hm_runs/records')['items'])
        label = {'user': uid, 'level': 1, 'segment': 1, 'class': 'soil'}
        created_label = request('/api/collections/hm_labels/records', 'POST', label, token=token)
        assert created_label['user'] == uid
        assert not request('/api/collections/hm_labels/records', token=token2)['items']
        assert len(request('/api/collections/hm_labels/records', token=token)['items']) == 1
        request('/api/collections/hm_labels/records', 'POST', label, token=token, expected=400)
        print('PASS PocketBase health, embedded web/WASM, signup, auth, runs, label privacy and uniqueness')
    finally:
        proc.terminate()
        proc.wait(timeout=10)
