UPDATE mastery_evidence
SET practice_context = CASE source_item_id
    WHEN 'lpic1.103.1.shell-environment-repair' THEN 'shell-env-repair'
    WHEN 'lpic1.103.1.transfer-shell-handoff' THEN 'shell-handoff'
    WHEN 'lpic1.103.5.stuck-worker' THEN 'process-incident'
    WHEN 'lpic1.103.5.transfer-operator-session' THEN 'operator-session'
    WHEN 'lpic1.104.5.shared-dropbox' THEN 'shared-dropbox-policy'
    WHEN 'lpic1.104.5.transfer-team-share-audit' THEN 'team-share-audit'
    ELSE practice_context
END
WHERE practice_context = ''
  AND source_item_id IN (
    'lpic1.103.1.shell-environment-repair',
    'lpic1.103.1.transfer-shell-handoff',
    'lpic1.103.5.stuck-worker',
    'lpic1.103.5.transfer-operator-session',
    'lpic1.104.5.shared-dropbox',
    'lpic1.104.5.transfer-team-share-audit'
  );
