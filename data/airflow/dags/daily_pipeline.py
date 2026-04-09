import os
from airflow import DAG
from airflow.providers.amazon.aws.operators.glue import GlueJobOperator
from airflow.providers.amazon.aws.operators.glue_crawler import GlueCrawlerOperator
from datetime import datetime, timedelta

# Captura das Envs (Configurações)
REGION = os.getenv('AWS_DEFAULT_REGION', 'us-east-1')
S3_RAW_BUCKET = os.getenv('S3_RAW_BUCKET')
GLUE_JOB_NAME = os.getenv('GLUE_JOB_NAME')
GLUE_CRAWLER_NAME = os.getenv('GLUE_CRAWLER_NAME')
GLUE_IAM_ROLE = os.getenv('GLUE_IAM_ROLE_NAME')
SCRIPT_S3_PATH = os.getenv('GLUE_SCRIPT_S3_PATH')

default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime(2023, 1, 1),
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
}

with DAG(
    'processamento_silver_diario',
    default_args=default_args,
    schedule_interval='*/10 * * * *',
    catchup=False
) as dag:

    run_glue_job = GlueJobOperator(
        task_id='run_glue_job',
        job_name=GLUE_JOB_NAME,
        script_location=SCRIPT_S3_PATH,
        s3_bucket=S3_RAW_BUCKET,
        iam_role_name=GLUE_IAM_ROLE,
        region_name=REGION,
    )

    run_crawler = GlueCrawlerOperator(
        task_id='run_glue_crawler',
        config={'Name': GLUE_CRAWLER_NAME},
        region_name=REGION,
    )

    run_glue_job >> run_crawler