"""Exercise a disposable PocketBase instance, migrations and ownership rules."""
import json
import os
import subprocess
import tempfile
import time
import urllib.error
import urllib.parse
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
        # Wins are replayed on the server (MAR-62), so these are real drives of the starter rover:
        # recorded inputs, and the time, samples, stars and score the game computes for them.
        design = json.loads('[{"x":0,"y":0,"k":"wheel"},{"x":1,"y":0,"k":"frame"},{"x":2,"y":0,"k":"frame"},{"x":3,"y":0,"k":"motor"},{"x":1,"y":1,"k":"frame"},{"x":2,"y":1,"k":"mast"}]')
        def win(user, level, inputs, time, score):
            return {'user': user, 'level': level, 'won': True, 'stars': 1, 'samples': 2, 'time': time,
                    'score': score, 'cost': 11, 'design': design, 'inputs': inputs}
        l1 = win(uid, 1, '/2QA/2QAkWQA', 24.6, 929)
        run = l1
        request('/api/collections/hm_runs/records', 'POST', run, expected=400)
        request('/api/collections/hm_runs/records', 'POST', run, token=token2, expected=400)
        created = request('/api/collections/hm_runs/records', 'POST', run, token=token)
        assert created['user'] == uid
        assert any(row['id'] == created['id'] for row in request('/api/collections/hm_runs/records')['items'])
        filtered = request('/api/collections/hm_runs/records?filter=' + urllib.parse.quote(f'user = "{uid}"') + '&fields=level,won,stars,time,score')
        assert filtered['totalItems'] == 1 and filtered['items'][0]['stars'] == 1 and filtered['items'][0]['time'] == 24.6
        label = {'user': uid, 'level': 1, 'segment': 1, 'class': 'soil'}
        created_label = request('/api/collections/hm_labels/records', 'POST', label, token=token)
        assert created_label['user'] == uid
        assert not request('/api/collections/hm_labels/records', token=token2)['items']
        assert len(request('/api/collections/hm_labels/records', token=token)['items']) == 1
        request('/api/collections/hm_labels/records', 'POST', label, token=token, expected=400)
        # Usernames and the leaderboard: public, usernames only, one profile per player.
        prof = {'user': uid, 'username': 'ann_9'}
        request('/api/collections/hm_profiles/records', 'POST', prof, expected=400)
        request('/api/collections/hm_profiles/records', 'POST', {'user': uid, 'username': 'Bad Name'}, token=token, expected=400)
        request('/api/collections/hm_profiles/records', 'POST', {'user': uid, 'username': 'ab'}, token=token, expected=400)
        request('/api/collections/hm_profiles/records', 'POST', {'user': uid, 'username': 'ann_9'}, token=token2, expected=400)
        mine = request('/api/collections/hm_profiles/records', 'POST', prof, token=token)
        request('/api/collections/hm_profiles/records', 'POST', {'user': uid, 'username': 'ann_10'}, token=token, expected=400)
        request('/api/collections/hm_profiles/records', 'POST', {'user': uid2, 'username': 'ANN_9'.lower()}, token=token2, expected=400)
        request('/api/collections/hm_profiles/records/' + mine['id'], 'PATCH', {'username': 'hijack'}, token=token2, expected=404)
        request('/api/collections/hm_profiles/records/' + mine['id'], 'PATCH', {'username': 'ann_99'}, token=token)
        # a second win on the same level with a lower score must not lower the best
        request('/api/collections/hm_runs/records', 'POST', win(uid, 2, '/2QA8WQA', 18.6, 1145), token=token)
        request('/api/collections/hm_runs/records', 'POST', win(uid, 2, '/1oA/FoA', 19.0, 1137), token=token)
        # forged: a perfect score with no drive behind it, and a real drive claiming a better score
        request('/api/collections/hm_runs/records', 'POST', dict(win(uid, 2, '', 5.0, 5000)), token=token, expected=400)
        request('/api/collections/hm_runs/records', 'POST', dict(win(uid, 2, '/1oA/FoA', 19.0, 1145)), token=token, expected=400)
        request('/api/collections/hm_runs/records', 'POST', {'user': uid, 'level': 1, 'won': False, 'stars': 0, 'time': 5, 'score': 0, 'cost': 11, 'design': design}, token=token)
        board = request('/api/collections/hm_leaderboard/records?sort=-score')
        assert board['totalItems'] == 1, board
        row = board['items'][0]
        assert row['username'] == 'ann_99' and row['score'] == 2074 and row['stars'] == 2 and row['missions'] == 2, row
        assert 'email' not in row and 'user' not in row
        # a player with no username is not on the board
        request('/api/collections/hm_runs/records', 'POST', win(uid2, 1, '/2QA/2QAkWQA', 24.6, 929), token=token2)
        assert request('/api/collections/hm_leaderboard/records')['totalItems'] == 1
        print('PASS usernames: validation, uniqueness, ownership; leaderboard: best per mission, public, usernames only')
        print('PASS PocketBase health, embedded web/WASM, signup, auth, runs, label privacy and uniqueness')
    finally:
        proc.terminate()
        proc.wait(timeout=10)
