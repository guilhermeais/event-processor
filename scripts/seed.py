import boto3
import json
import os


schemas_table_name = os.environ.get('SCHEMAS_TABLE_NAME')
endpoint_url = os.environ.get('AWS_ENDPOINT_URL', 'http://localhost:4566')

if not schemas_table_name:
    raise ValueError("A variável de ambiente SCHEMAS_TABLE_NAME não foi definida!")

print(f"Conectando ao DynamoDB em: {endpoint_url}")
print(f"Tabela alvo: {schemas_table_name}")

dynamodb = boto3.resource(
    'dynamodb',
    endpoint_url=endpoint_url
)

table = dynamodb.Table(schemas_table_name)
file_path = 'scripts/default-schemas.json'

try:
    with open(file_path, 'r') as f:
        schemas = json.load(f)

    for schema_data in schemas:
        event_type = schema_data['event_type']
        schema_json_string = json.dumps(schema_data['schema'])

        print(f"Inserindo schema para event_type: {event_type}")
        table.put_item(
            Item={
                'event_type': event_type,
                'schema': schema_json_string
            }
        )
        
    print("Dados inseridos com sucesso!")

except Exception as e:
    print(f"Erro ao inserir dados: {e}")