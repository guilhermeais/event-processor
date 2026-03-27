# Event Processor

**Português** | **[English](README.en.md)**

---

Este projeto é a implementação de um componente **Event Processor** para uma plataforma de dados. O objetivo é construir um serviço reativo focado em consumir eventos de uma mensageria, realizar a validação contra contratos pré-definidos e fazer a triagem. O resultado final deve garantir baixa latência para o consumo por serviços subsequentes, operando de forma resiliente em um ecossistema multi-tenant.

![O problema a ser resolvido](./docs/architecture.png)

---

## 📑 Índice

1. [🏗️ Arquitetura e Decisões Técnicas](#️-1-arquitetura-e-decisões-técnicas)
   - [Diagrama da Solução](#diagrama-da-solução)
   - [Justificativa de Algumas Escolhas](#justificativa-de-algumas-escolhas)
2. [💻 Design de Código (Arquitetura Hexagonal)](#-2-design-de-código-arquitetura-hexagonal)
3. [📋 Roadmap de Implementação](#-3-roadmap-de-implementação)
   - [Requisitos Funcionais](#requisitos-funcionais)
   - [Requisitos Não Funcionais](#requisitos-não-funcionais)
4. [🚀 Como Executar](#-4-como-executar)
   - [Passo a Passo](#passo-a-passo)
   - [Pontos de evolução](#pontos-de-evolução)
5. [🧪 Testes e Qualidade](#-5-testes-e-qualidade)

---

## 🏗️ 1. Arquitetura e Decisões Técnicas

Para atender aos requisitos de escalabilidade e resiliência, a seguinte stack foi definida para simulação local:
### Diagrama da Solução

![Arquitetura do Event Processor](./docs/proposed-architecture.png)

### Justificativa de Algumas Escolhas

* **Mensageria (AWS SQS):** Escolhido para garantir que nenhum evento seja perdido em caso de falhas. O SQS atuará com o a flila de eventos e será configurado com uma Dead Letter Queue (DLQ) para mensagens que estiverem inválidas ou com algum problema no processamento. 
  * Eu havia considerado o uso do Kafka, mas optei pelo SQS pelos seguintes motivos: 
    * Integração nativa (*Event Source Mapping*) com o AWS Lambda. 
    * Como a arquitetura proposta não exige a garantia de ordem estrita de processamento ou *replay* histórico de eventos, o SQS atende perfeitamente ao requisito de mensageria com resiliência sem a sobrecarga de gerenciar *brokers*, partições ou *offsets* exigida pelo Kafka.
* **Processamento (AWS Lambda em Go):** A arquitetura utilizará o modelo *Event-Driven* com AWS Lambda (linguagem de progamação Go) acionada nativamente pelo SQS. 
  * *Por que não Kubernetes com HPA?* Embora rodar a aplicação em pods dentro de um cluster Kubernetes e escalá-los horizontalmente (HPA) observando a profundidade da fila e escalando de acordo seja uma abordagem robusta, ela introduz uma complexidade operacional significativa (gerenciamento de cluster, manifestos, e etc...). A AWS Lambda oferece escalabilidade elástica gerida pela própria nuvem, instanciando novos ambientes de execução instantaneamente conforme o volume da fila aumenta. Isso atende perfeitamente ao requisito de criar um serviço reativo de alta performance, priorizando a **simplicidade**, mesmo que essa arquitetura não seja, de primeiro momento, a mais barata.
* **Validação (JSON Schema):** Para atender à regra de contratos declarativos, os *schemas* em JSON serão cacheados em memória durante a inicialização da Lambda, validando o *payload* de forma extremamente rápida. (Os schemas serão carregados através de uma tablea no DynamoDb contendo o `event_type` e o `schema` em si).
    - Outras formas de validar seriam, por exemplo, o uso de Protobuf (Protocol Buffers), validação estrutural no código via *struct tags*, porém, escolhi a JSON Schema pois permite especificar os eventos de forma declarativa e ser usada como base para a validação. Além de ser **agnóstica de linguagem e nativamente suportada por payloads JSON (padrão de mercado)**, ela permite adicionar novos produtores e regras apenas incluindo novos registros na tabela de contrato (`schemas), sem a necessidade de recompilar a aplicação.
* **Persistência (AWS DynamoDB):** Banco NoSQL que escala horizontalmente por natureza. A modelagem usará a `Partition Key` para agrupar dados por `client_id` (multi-tenant) e `Sort Key` com id único do evento para garantir a idempotência das mensagens (o ideal é trabalhemos com algum uuid sequencial aqui, como por exemplo, `uuidv7` que usei no `producer`). O uso do *DynamoDB Streams* atuará como um CDC (Change Data Capture), habilitando o consumo de baixa latência por outros serviços (Sender).
  * Eu havia considerado, inicialmente, PostgresSQL ou CassandraDB, porém:
  * *Por que não PostgreSQL?* Embora o Postgres consiga lidar com os payloads dinâmicos usando o tipo `jsonb`, bancos relacionais enfrentam gargalos de escalabilidade horizontal em cenários de alta ingestão (*write-heavy*) e exigiriam um gerenciamento complexo de pool de conexões via Lambda.
  * *Por que não CassandraDB?* O Cassandra lidaria perfeitamente com o alto volume de escritas e a escalabilidade, porém adicionaria uma complexidade operacional considerável (manutenção de cluster e tuning) que vai contra a premissa de simplicidade. O DynamoDB entrega a mesma performance no modelo *Serverless*.
  * TO-DO: Citar/estudar a importância do TTL para as mensagens. Dependendo da quantia de eventos, manter todos persistidos vai ocupar muito disco. Talvez, como já disponibilizaremos os eventos com DynamoDB Streams, valha a pena colocar um TTL nas mensagens e, talvez, movê-los pra um S3 para permanência.
* **Observabilidade (Logs, Métricas e Traces):** 
  * *Logs:* Adotei o padrão de **Structured Logging**. Em vez de múltiplos logs sequenciais no I/O, a aplicação consolida o contexto da execução e emite um único payload JSON por evento processado (contendo `event_id`, `client_id`, tempo de execução e status, etc..). Tomei essa decisão após ler este blog e concordar com a visão do mesmo: [Logging Sucks](https://loggingsucks.com/). O destino nativo é o **Amazon CloudWatch**, já que logaremos no stdout das AWS Lambdas.
  * *Métricas:* A própria Lambda nos dá métricas/alertas de invocações, erros e duração. O SQS também nos entrega o `Queue Depth`.
  * *Tracing:* Como nosso fluxo de processamento já é `SQS -> Lambda -> DynamoDB`, usaria o **AWS X-Ray** para o rastreamento distribuído e a visualização do mapa de serviço completo para debugar gargalos de performance. Esse foi mais dificil de simular localmente pois o localstack exige o plano `pro` para visualiza-lo.
  
  Não é o ideal, poderíamos fazer o uso aqui de um OTEL para métricas e tracing, por exemplo, porém, dado o tempo de entrega do projeto, optei pelo simples
* **Roteamento (Sender):** Para o serviço subsequente (`Sender`), não estava no escopo fazer a implementação dele aqui, mas acho válido comentar uma estrategia de entrega desses eventos em um cenário multi-tenant. Como o `Sender` precisa entregar as mensagens aos múltiplos clientes finais de forma isolada, a estratégia que eu adotaria, seria **Fan-out com Amazon SNS + Message Filtering**. Usaríamos o `EventBridge` para capturar o DynamoDB Streams e publicaria todos os eventos em um único tópico SNS. Cada cliente teria uma fila SQS exclusiva inscrita neste tópico, contendo uma *Filter Policy* apontando para o seu respectivo `client_id`.
![Arquitetura Sender](./docs/sender-arch.png)
  
---
## 💻 2. Design de Código (Arquitetura Hexagonal)

![Design de Código do Event Processor](./docs/code-design.png)

Para garantir a qualidade do software e proteger a regra de negócios, utilizei conceitos de **Arquitetura Hexagonal (Ports and Adapters)**. Essa decisão visa garantir o baixo acoplamento, priorizando a manutenibilidade e a facilidade de criação de testes. 

A estrutura da aplicação foi dividida da seguinte forma:



* **Portas de Entrada (Driving Adapters):** Representado pelo *entrypoint* da aplicação (o *handler* da AWS Lambda consumindo o SQS). Optei por não criar abstrações excessivamente complexas aqui em prol da simplicidad; o papel desse *handler* é estritamente receber o *payload* da AWS, convertê-lo para o modelo de domínio e acionar a regra de negócio.
* **Core / Regras de Negócio (Use Case):** A inteligência da aplicação está isolada no caso de uso `Processor`. Ele é o maestro do fluxo: recebe a mensagem, interage com as portas de saída para validar o contrato e, se a mensagem for válida, aciona a persistência. O `Processor` não conhece bibliotecas da AWS ou de validação; ele conhece apenas *Interfaces* nativas do Go.
* **Portas de Saída (Driven Adapters):** São as implementações concretas dos contratos exigidos pelo Core. 
  * *Validator:* Implementação concreta utilizando a engine de JSON Schema.
  * *Persister:* Implementação concreta utilizando o AWS SDK v2 para o DynamoDB.

Assim protegemos o *Core* da aplicação e podemos ter uma **testabilidade facilitada*. Durante a criação dos testes do `Processor`, injetei *mocks* do validador e do banco de dados, testando todas as ramificações de erro e sucesso sem precisar de infraestrutura real. Além disso, a aplicação se torna altamente extensível: se amanhã a persistência mudar do DynamoDB para outro banco, basta injetar um novo adaptador, sem alterar uma única linha da regra de negócio.

---
## 📋 3. Roadmap de Implementação


### Requisitos Funcionais
- [x] **Consumo:** O serviço deve consumir eventos de uma fila SQS.
- [x] **Roteamento por Tipo:** Distinguir eventos utilizando um identificador de tipo (`event_type`).
- [x] **Validação Declarativa:** Validar os eventos recebidos contra os contratos.
- [x] **Persistência Preparatória:** Salvar o evento no DynamoDB.

### Requisitos Não Funcionais
- [x] **Resiliência:** Implementação de DLQ para garantir zero perda de eventos inválidos ou com falha.
- [x] **Testabilidade:** Criação de testes unitários e de integração (integração com os componentes usando LocalStack).
- [x] **Reprodutibilidade:** Criação de um ambiente Docker/LocalStack com script de infraestrutura para facilitar a validação.

## 🚀 4. Como Executar

Para garantir que o ambiente seja facilmente replicado, toda a orquestração de infraestrutura, *build* e execução foi encapsulada no `Makefile`. 

**Pré-requisitos:**
* Docker e Docker Compose instalados.
* Go 1.25+ (para rodar os testes e o produtor de eventos).
* `make` e `zip` (para o empacotamento da Lambda).

> Recomendo rodar em um ambiente `linux`.

### Passo a Passo

**1. Subir a Infraestrutura (LocalStack + Lambda):**

O comando abaixo irá compilar a aplicação Go (garantindo compatibilidade com o ambiente Linux da AWS Lambda usando um container Docker), empacotar o binário e subir o LocalStack já provisionando a fila SQS, a tabela no DynamoDB e fazendo o *deploy* da Lambda.
O provisionamento recursos é feito usando IAC com Terraform. Dentro da pasta `terraform/`, temos o IAC que provisiona a infraestrutura dentro do localstack.
```bash
make up
```

**2. Injetar Eventos (Producer):**

Com a infraestrutura no ar, você pode gerar eventos simulados e publicá-los na fila SQS executando o mock producer:
```bash
make run-producer
```
Ele ira gerar 5.000 eventos aleatórios com os diferentes schemas padrões (`ACCOUNT_CREATED`, `TRANSACTION_AUTHORIZED`, `CARD_ISSUED` e `CREDIT_ANALYSIS_APPROVED`) e enviar pra fila criada no `make up`.

**3. Acompanhar a Observabilidade (Logs):**

Para visualizar o resultado do processamento reativo (o SQS acionando a Lambda) e ver o Structured Logging em ação através do CloudWatch do LocalStack, execute:
```bash
make logs-cw
```
Caso precise debugar a infraestrutura em si, você pode usar `make logs` para ver a saída do container do LocalStack.

**4. Derrumar Ambiente**

Para parar os containers, limpar os volumes (banco e fila) e remover a pasta de build, execute:
```bash
make down
```

#### Pontos de evolução
- Invés de ter o script `localstack/init-aws.sh` que roda dentro do container localstack e usa o `awslocal` para criar a arquitetura necessária pra aplicação rodar, eu teria utilizado algum mecanimos de IAC mais moderno, como por exemplo, `terraform`. Sei que o localstack é compatível com ele [de acordo com essa documentação](https://docs.localstack.cloud/aws/capabilities/config/initialization-hooks/#terraform-files-as-init-hooks). Não o usei pois o tempo para o projeto era relativamente curto, e também, preciso estudar e melhorar minhas habilidades utilizando o mesmo.

## 🧪 5. Testes e Qualidade
O projeto conta com testes de integração focados em garantir a testabilidade do Core (regra de negócio). Quando falo de teste de integração, digo componentes que interagem entre si. A minha premissa é que, por mais que eu crie um `mock/stub` e injete ele em um caso de uso, o caso de uso **integra** com outras partes, injetar um mock não o torna teste unitário.

Dito isso, teve momentos que fiz o uso de `mocks/stubs` para testar - como por exemplo, o caso de uso `Processor` -, como teve momentos que subi, de fato, um `localstack` usando `testcontainers` para validar integração com dynamodb (Ex: `tests/internal/infra/persister/dynamodb_persister_test.go` e `tests/internal/infra/validator/jsonschema_loader_test.go`)

**1. Para rodar a suíte completa de testes:**

```bash
make test
```

**2. Para gerar e visualizar o relatório de cobertura de código:**

```bash
make coverage-html
```

Para os testes, prefiri subir o `localstack` diretamente do código, facilitando o setup inicial para rodar os testes. Você pode consultar os detalhes dessa implementação em um utilitário de teste no arquivo `tests/testhelpers/localstack_helper.go`