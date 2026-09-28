ALTER TABLE notification_delivery
ADD COLUMN status TEXT NOT NULL DEFAULT 'sent'
CHECK (status IN ('claimed', 'sent'));
