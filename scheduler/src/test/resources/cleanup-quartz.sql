-- Clean up Quartz tables to prevent orphaned trigger issues
DELETE FROM qrtz_simple_triggers;
DELETE FROM qrtz_simprop_triggers;
DELETE FROM qrtz_cron_triggers;
DELETE FROM qrtz_blob_triggers;
DELETE FROM qrtz_triggers;
DELETE FROM qrtz_job_details;
DELETE FROM qrtz_fired_triggers;
