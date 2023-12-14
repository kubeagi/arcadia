-- Table: public.data_process_task

-- DROP TABLE IF EXISTS public.data_process_task;

CREATE TABLE IF NOT EXISTS public.data_process_task
(
    id character varying(32) COLLATE pg_catalog."default" NOT NULL,
    name character varying(64) COLLATE pg_catalog."default",
    file_type character varying(32) COLLATE pg_catalog."default",
    status character varying(32) COLLATE pg_catalog."default",
    pre_data_set_name character varying(32) COLLATE pg_catalog."default",
    pre_data_set_version character varying(32) COLLATE pg_catalog."default",
    file_names jsonb,
    post_data_set_name character varying(32) COLLATE pg_catalog."default",
    post_data_set_version character varying(32) COLLATE pg_catalog."default",
    data_process_config_info jsonb,
    start_datetime character varying(32) COLLATE pg_catalog."default",
    end_datetime character varying(32) COLLATE pg_catalog."default",
    create_datetime character varying(32) COLLATE pg_catalog."default",
    create_user character varying(32) COLLATE pg_catalog."default",
    create_program character varying(64) COLLATE pg_catalog."default",
    update_datetime character varying(32) COLLATE pg_catalog."default",
    update_user character varying(32) COLLATE pg_catalog."default",
    update_program character varying(64) COLLATE pg_catalog."default",
    namespace character varying(64) COLLATE pg_catalog."default",
    CONSTRAINT data_process_task_pkey PRIMARY KEY (id)
)

-- Table: public.data_process_task_detail

-- DROP TABLE IF EXISTS public.data_process_task_detail;

CREATE TABLE IF NOT EXISTS public.data_process_task_detail
(
    id character varying(32) COLLATE pg_catalog."default" NOT NULL,
    task_id character varying(32) COLLATE pg_catalog."default",
    file_name character varying(512) COLLATE pg_catalog."default",
    transform_type character varying(64) COLLATE pg_catalog."default",
    pre_content text COLLATE pg_catalog."default",
    post_content text COLLATE pg_catalog."default",
    create_datetime character varying(32) COLLATE pg_catalog."default",
    create_user character varying(32) COLLATE pg_catalog."default",
    create_program character varying(64) COLLATE pg_catalog."default",
    update_datetime character varying(32) COLLATE pg_catalog."default",
    update_user character varying(32) COLLATE pg_catalog."default",
    update_program character varying(32) COLLATE pg_catalog."default",
    CONSTRAINT data_process_detail_pkey PRIMARY KEY (id)
)

COMMENT ON TABLE public.data_process_task_detail IS '数据处理详情';
COMMENT ON COLUMN public.data_process_task_detail.id IS '主键';
COMMENT ON COLUMN public.data_process_task_detail.task_id IS '任务Id';
COMMENT ON COLUMN public.data_process_task_detail.file_name IS '文件名称';
COMMENT ON COLUMN public.data_process_task_detail.transform_type IS '转换类型';
COMMENT ON COLUMN public.data_process_task_detail.pre_content IS '处理前的内容';
COMMENT ON COLUMN public.data_process_task_detail.post_content IS '处理后的内容';
COMMENT ON COLUMN public.data_process_task_detail.create_datetime IS '创建时间';
COMMENT ON COLUMN public.data_process_task_detail.create_user IS '创建用户';
COMMENT ON COLUMN public.data_process_task_detail.create_program IS '创建程序';
COMMENT ON COLUMN public.data_process_task_detail.update_datetime IS '更新时间';
COMMENT ON COLUMN public.data_process_task_detail.update_user IS '更新用户';
COMMENT ON COLUMN public.data_process_task_detail.update_program IS '更新程序';

CREATE TABLE public.data_process_task_question_answer (
	id varchar(32) NOT NULL, -- 主键
	task_id varchar(32) NULL, -- 任务Id
	file_name varchar(512) NULL, -- 文件名称
	question text NULL, -- 问题
	answer text NULL, -- 答案
	create_datetime varchar(32) NULL, -- 创建时间
	create_user varchar(32) NULL, -- 创建用户
	create_program varchar(64) NULL, -- 创建程序
	update_datetime varchar(32) NULL, -- 更新时间
	update_user varchar(32) NULL, -- 更新用户
	update_program varchar(32) NULL, -- 更新程序
	CONSTRAINT data_process_task_question_answer_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE public.data_process_task_question_answer IS '数据处理问题答案';

-- Column comments

COMMENT ON COLUMN public.data_process_task_question_answer.id IS '主键';
COMMENT ON COLUMN public.data_process_task_question_answer.task_id IS '任务Id';
COMMENT ON COLUMN public.data_process_task_question_answer.file_name IS '文件名称';
COMMENT ON COLUMN public.data_process_task_question_answer.question IS '问题';
COMMENT ON COLUMN public.data_process_task_question_answer.answer IS '答案';
COMMENT ON COLUMN public.data_process_task_question_answer.create_datetime IS '创建时间';
COMMENT ON COLUMN public.data_process_task_question_answer.create_user IS '创建用户';
COMMENT ON COLUMN public.data_process_task_question_answer.create_program IS '创建程序';
COMMENT ON COLUMN public.data_process_task_question_answer.update_datetime IS '更新时间';
COMMENT ON COLUMN public.data_process_task_question_answer.update_user IS '更新用户';
COMMENT ON COLUMN public.data_process_task_question_answer.update_program IS '更新程序';

