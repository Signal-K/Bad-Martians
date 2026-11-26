#!/usr/bin/env python3
"""Beta metrics straight from PocketBase, no admin login needed.

hm_runs, hm_profiles and hm_leaderboard are publicly readable (the leaderboard
needs it), so this works against production with just the API URL:

    HM_API_URL=https://signal-k-pocketbase.fly.dev scripts/beta-metrics.py [--days 7]

It answers, from the game's own records: how many players, how many reach a
first win, win rate by mission, how many come back on a later day, how many
set a username. It cannot see anonymous visitors who never signed in; PostHog
has those. Players who play without an account are invisible here.
"""
import argparse, collections, datetime as dt, json, os, sys, urllib.parse, urllib.request

ap = argparse.ArgumentParser()
ap.add_argument('--api', default=os.environ.get('HM_API_URL', ''))
ap.add_argument('--days', type=int, default=7, help='window for "recent" figures')
a = ap.parse_args()
if not a.api:
    sys.exit('Set HM_API_URL (or pass --api).')

def get(path, **q):
    url = a.api.rstrip('/') + path + ('?' + urllib.parse.urlencode(q) if q else '')
    with urllib.request.urlopen(url, timeout=20) as r:
        return json.load(r)

def pages(col, fields):
    page = 1
    while True:
        d = get(f'/api/collections/{col}/records', page=page, perPage=500, fields=fields, sort='created')
        yield from d['items']
        if page >= d['totalPages']:
            return
        page += 1

runs = list(pages('hm_runs', 'user,level,won,stars,created'))
profiles = get('/api/collections/hm_profiles/records', perPage=1)['totalItems']
if not runs:
    print('No runs yet.'); sys.exit(0)

day = lambda r: r['created'][:10]
players = collections.defaultdict(list)
for r in runs:
    players[r['user']].append(r)
won = {u for u, rs in players.items() if any(r['won'] for r in rs)}
returning = {u for u, rs in players.items() if len({day(r) for r in rs}) > 1}
cut = (dt.date.today() - dt.timedelta(days=a.days)).isoformat()
recent = {u for u, rs in players.items() if any(day(r) >= cut for r in rs)}

def pct(n, d): return f'{100*n/d:.0f}%' if d else 'n/a'
print(f'Signed-in players with a run : {len(players)}')
print(f'  reached a first win        : {len(won)} ({pct(len(won), len(players))})')
print(f'  played on 2+ different days: {len(returning)} ({pct(len(returning), len(players))})')
print(f'  active in last {a.days} days        : {len(recent)}')
print(f'  set a username             : {profiles} ({pct(profiles, len(players))})')
print(f'Runs: {len(runs)} total, {sum(1 for r in runs if r["won"])} won')
print('\nWin rate by mission (hardest first; attempts, wins):')
by = collections.defaultdict(lambda: [0, 0])
for r in runs:
    by[r['level']][0] += 1
    by[r['level']][1] += bool(r['won'])
for lvl, (n, w) in sorted(by.items(), key=lambda kv: kv[1][1] / kv[1][0]):
    print(f'  mission {lvl:>4}: {pct(w, n):>4}  ({w}/{n})')
print('\nPlayers by furthest mission won:')
far = collections.Counter(max((r['level'] for r in rs if r['won']), default=0) for rs in players.values())
for lvl in sorted(far):
    print(f'  {"no wins" if lvl == 0 else "mission %d" % lvl:>10}: {far[lvl]}')
