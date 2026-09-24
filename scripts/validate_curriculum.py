#!/usr/bin/env python3
from pathlib import Path
import json, sys
p=Path(__file__).resolve().parents[1]/'curriculum/lpic-1-v5/objectives.json'
data=json.loads(p.read_text(encoding='utf-8'))
objs=data['objectives']
errors=[]
ids=[o['id'] for o in objs]
if len(ids)!=len(set(ids)): errors.append('duplicate objective IDs')
expected_counts={'101':23,'102':19}
expected_weights={'101':60,'102':60}
for exam in ('101','102'):
    rows=[o for o in objs if o['exam']==exam and o.get('active')]
    if len(rows)!=expected_counts[exam]: errors.append(f'exam {exam}: expected {expected_counts[exam]} active objectives, got {len(rows)}')
    weight=sum(o['weight'] for o in rows)
    if weight!=expected_weights[exam]: errors.append(f'exam {exam}: expected weight 60, got {weight}')
for o in objs:
    for field in ('id','exam','topic','weight','title_fr','concepts','terms_files_utilities','recommended_backend','assessment_evidence','source'):
        if field not in o or o[field] in ('',[],None): errors.append(f"{o.get('id','?')}: missing {field}")
    if not str(o['id']).startswith(o['topic']+'.'): errors.append(f"{o['id']}: topic mismatch")
if '104.4' in ids: errors.append('104.4 must not be active in LPIC-1 v5 inventory')
if errors:
    print('Curriculum validation FAILED:')
    for e in errors: print(' -',e)
    sys.exit(1)
print(f"Curriculum validation OK: {len(objs)} active objectives; exam 101 weight=60; exam 102 weight=60")
