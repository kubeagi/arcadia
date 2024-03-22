ALTER TABLE data_process_task_log ADD COLUMN exc_msg text;

COMMENT ON COLUMN data_process_task_log.exc_msg IS '异常信息';