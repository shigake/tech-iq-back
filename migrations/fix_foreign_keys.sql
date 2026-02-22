-- Migration: Fix foreign keys to allow user deletion
-- Run this on production database before deploying the new code

-- Drop existing foreign key constraints
ALTER TABLE activity_logs DROP CONSTRAINT IF EXISTS fk_activity_logs_user;
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS fk_audit_logs_user;
ALTER TABLE security_logs DROP CONSTRAINT IF EXISTS fk_security_logs_user;
ALTER TABLE financial_entries DROP CONSTRAINT IF EXISTS fk_financial_entries_created_by_user;
ALTER TABLE financial_entries DROP CONSTRAINT IF EXISTS fk_financial_entries_updated_by_user;
ALTER TABLE payment_batches DROP CONSTRAINT IF EXISTS fk_payment_batches_created_by_user;
ALTER TABLE payment_batches DROP CONSTRAINT IF EXISTS fk_payment_batches_approved_by_user;

-- Alter columns to allow NULL where needed
ALTER TABLE activity_logs ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE audit_logs ALTER COLUMN user_id DROP NOT NULL;

-- Recreate foreign keys with ON DELETE SET NULL
ALTER TABLE activity_logs 
ADD CONSTRAINT fk_activity_logs_user 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE audit_logs 
ADD CONSTRAINT fk_audit_logs_user 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE security_logs 
ADD CONSTRAINT fk_security_logs_user 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE financial_entries 
ADD CONSTRAINT fk_financial_entries_created_by_user 
FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE financial_entries 
ADD CONSTRAINT fk_financial_entries_updated_by_user 
FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE payment_batches 
ADD CONSTRAINT fk_payment_batches_created_by_user 
FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE payment_batches 
ADD CONSTRAINT fk_payment_batches_approved_by_user 
FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL;

-- Verify the changes
SELECT tc.table_name, tc.constraint_name, kcu.column_name, rc.delete_rule
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu 
    ON tc.constraint_name = kcu.constraint_name
LEFT JOIN information_schema.referential_constraints rc 
    ON tc.constraint_name = rc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY' 
    AND kcu.column_name IN ('user_id', 'created_by', 'updated_by', 'approved_by')
ORDER BY tc.table_name;
