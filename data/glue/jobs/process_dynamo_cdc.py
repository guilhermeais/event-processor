import sys
from awsglue.utils import getResolvedOptions
from pyspark.context import SparkContext
from awsglue.context import GlueContext
from awsglue.job import Job
from pyspark.sql.functions import col, row_number, from_unixtime, to_timestamp, year, month, dayofmonth
from pyspark.sql.window import Window

args = getResolvedOptions(sys.argv, ['JOB_NAME', 'S3_RAW_PATH', 'S3_SILVER_PATH'])

sc = SparkContext()
glueContext = GlueContext(sc)
spark = glueContext.spark_session
job = Job(glueContext)
job.init(args['JOB_NAME'], args)

print(f"Lendo dados da camada Raw: {args['S3_RAW_PATH']}")
df_raw = spark.read.json(args['S3_RAW_PATH'])

df_cleaned = df_raw.select(
    col("dynamodb.NewImage.id.S").alias("id"), 
    col("dynamodb.NewImage.client_id.S").alias("client_id"), 
    col("dynamodb.NewImage.event_id.S").alias("event_id"),
    col("dynamodb.NewImage.payload.S").alias("payload"),
    col("dynamodb.NewImage.event_type.S").alias("event_type"),
    col("eventName").alias("event_name"),
    col("dynamodb.ApproximateCreationDateTime").alias("event_timestamp"),
)

df_cleaned = df_cleaned.withColumn(
    "event_date",
    to_timestamp(from_unixtime(col("event_timestamp").cast("double") / 1000.0)),
)

window_spec = Window.partitionBy("id").orderBy(col("event_timestamp").desc())

df_latest_state = df_cleaned.withColumn("row_num", row_number().over(window_spec)) \
                            .filter(col("row_num") == 1) \
                            .drop("row_num")


df_final = df_latest_state.filter(col("event_name") != "REMOVE")

df_final = df_final.withColumn("year", year("event_date")) \
                   .withColumn("month", month("event_date")) \
                   .withColumn("day", dayofmonth("event_date"))

print(f"Escrevendo dados na camada Silver: {args['S3_SILVER_PATH']}")

df_final.write \
    .mode("overwrite") \
    .partitionBy("year", "month", "day") \
    .parquet(args['S3_SILVER_PATH'])

job.commit()