# Rate Limiter em Go com Redis

Este projeto implementa um **Rate Limiter em Go** como middleware para uma aplicação HTTP, utilizando **Redis** para armazenar o estado dos limites de requisições.

A ideia principal é limitar o número de requisições por segundo com base no **IP do solicitante** ou em um **token enviado no header `API_KEY`**.

## Regras implementadas

- Limite por IP: **10 requisições por segundo**
- Limite padrão por token: **10 requisições por segundo**
- `token-basic`: **20 requisições por segundo**
- `token-premium`: **100 requisições por segundo**
- O token sempre tem **prioridade sobre o IP**
- Ao ultrapassar o limite, a API retorna **HTTP 429**
- O cliente infrator fica bloqueado por um período configurável
- O Redis é utilizado para armazenar o estado do Rate Limiter
- A camada de persistência é desacoplada por meio de uma interface
- A implementação de persistência pode ser substituída futuramente sem alterar a lógica principal do Rate Limiter
- As configurações são definidas por variáveis de ambiente
- O projeto pode ser executado e testado utilizando Docker e Docker Compose

## Tecnologias

- Go
- Redis
- Docker
- Docker Compose
- HTTP Middleware
- Lua para operações atômicas no Redis
- Testes automatizados com `go test`

## Arquitetura

O fluxo principal da aplicação é:

```text
Requisição HTTP
      |
      v
Rate Limit Middleware
      |
      v
RateLimiter
      |
      v
RateStore (interface)
      |
      v
RedisStore
      |
      v
Redis
```

O middleware é responsável por interceptar as requisições e utilizar a lógica do `RateLimiter`.

A regra de negócio do Rate Limiter não fica diretamente dentro do middleware.

A interface `RateStore` desacopla a lógica de negócio da implementação de persistência. Dessa forma, é possível substituir o `RedisStore` por outra implementação no futuro sem alterar a lógica principal do Rate Limiter.

Atualmente, a implementação utilizada é o `RedisStore`.

## Estrutura do projeto

Uma visão simplificada do projeto:

```text
rate-limiter-go/

├── cmd/
│   └── server/
│
├── internal/
│   ├── config/
│   ├── limiter/
│   ├── middleware/
│   └── store/
│
├── Dockerfile
├── docker-compose.yaml
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

## Configuração

As configurações da aplicação são definidas por variáveis de ambiente.

O projeto possui um arquivo `.env.example` como modelo para criação do `.env`.

### Criar o `.env`

No PowerShell:

```powershell
Copy-Item .env.example .env
```

No Linux/macOS:

```bash
cp .env.example .env
```

### Exemplo de configuração

```env
APP_PORT=8080

RATE_LIMIT_IP=10
RATE_LIMIT_TOKEN_DEFAULT=10

TOKEN_LIMITS_JSON={"token-premium":100,"token-basic":20}

BLOCK_DURATION_SECONDS=300

REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0

RATE_LIMIT_WINDOW_SECONDS=1
```

### Principais configurações

| Variável                    | Descrição                               |
| --------------------------- | --------------------------------------- |
| `APP_PORT`                  | Porta utilizada pela aplicação          |
| `RATE_LIMIT_IP`             | Limite de requisições por IP            |
| `RATE_LIMIT_TOKEN_DEFAULT`  | Limite padrão para tokens               |
| `TOKEN_LIMITS_JSON`         | Limites específicos para tokens         |
| `BLOCK_DURATION_SECONDS`    | Tempo de bloqueio após exceder o limite |
| `REDIS_ADDR`                | Endereço do Redis                       |
| `REDIS_PASSWORD`            | Senha do Redis, caso configurada        |
| `REDIS_DB`                  | Banco utilizado no Redis                |
| `RATE_LIMIT_WINDOW_SECONDS` | Janela de controle das requisições      |

Não é necessário alterar o código para modificar os limites. Basta alterar as variáveis de ambiente.

## Executando com Docker

O projeto foi preparado para funcionar utilizando Docker Compose.

### Subir a aplicação

Execute:

```powershell
docker compose up --build
```

A aplicação ficará disponível em:

```text
http://localhost:8080/
```

Para verificar se a aplicação está funcionando:

```powershell
Invoke-WebRequest http://localhost:8080/ -UseBasicParsing
```

A resposta esperada é:

```text
rate limiter is running
```

## Testes manuais

Os testes abaixo demonstram o funcionamento do Rate Limiter em diferentes cenários.

### Teste manual por IP

Primeiro, limpe o Redis para começar o teste do zero:

```powershell
docker compose exec redis redis-cli FLUSHDB
```

Depois, envie 12 requisições sem o header `API_KEY`:

```powershell
1..12 | ForEach-Object {
    try {
        $r = Invoke-WebRequest `
            http://localhost:8080/ `
            -UseBasicParsing

        Write-Host "$_ -> $($r.StatusCode)"
    }
    catch {
        Write-Host "$_ -> $($_.Exception.Response.StatusCode.value__)"
    }
}
```

Com o limite padrão de 10 requisições por segundo, o resultado esperado é:

```text
1 -> 200
2 -> 200
...
10 -> 200
11 -> 429
12 -> 429
```

Isso demonstra que o limite por IP está funcionando.

## Teste manual com token

Para testar o `token-premium`, primeiro limpe o Redis:

```powershell
docker compose exec redis redis-cli FLUSHDB
```

Depois, defina o token:

```powershell
$headers = @{ API_KEY = "token-premium" }
```

Envie 12 requisições:

```powershell
1..12 | ForEach-Object {
    try {
        $r = Invoke-WebRequest `
            http://localhost:8080/ `
            -Headers $headers `
            -UseBasicParsing

        Write-Host "$_ -> $($r.StatusCode)"
    }
    catch {
        Write-Host "$_ -> $($_.Exception.Response.StatusCode.value__)"
    }
}
```

Como o `token-premium` possui limite de 100 requisições por segundo, as 12 requisições devem retornar:

```text
1 -> 200
2 -> 200
...
12 -> 200
```

## Teste de precedência: Token > IP

Esse teste demonstra que o token possui prioridade sobre o limite do IP.

Primeiro, limpe o Redis:

```powershell
docker compose exec redis redis-cli FLUSHDB
```

Agora, envie 12 requisições sem token:

```powershell
1..12 | ForEach-Object {
    try {
        $r = Invoke-WebRequest `
            http://localhost:8080/ `
            -UseBasicParsing

        Write-Host "$_ -> $($r.StatusCode)"
    }
    catch {
        Write-Host "$_ -> $($_.Exception.Response.StatusCode.value__)"
    }
}
```

As primeiras 10 requisições devem retornar `200` e as duas últimas devem retornar `429`:

```text
1 -> 200
2 -> 200
...
10 -> 200
11 -> 429
12 -> 429
```

**Não limpe o Redis novamente neste momento.**

Agora utilize o `token-premium`:

```powershell
$headers = @{ API_KEY = "token-premium" }
```

Envie outras 12 requisições:

```powershell
1..12 | ForEach-Object {
    try {
        $r = Invoke-WebRequest `
            http://localhost:8080/ `
            -Headers $headers `
            -UseBasicParsing

        Write-Host "$_ -> $($r.StatusCode)"
    }
    catch {
        Write-Host "$_ -> $($_.Exception.Response.StatusCode.value__)"
    }
}
```

O resultado esperado é:

```text
1 -> 200
2 -> 200
...
12 -> 200
```

Isso demonstra que, mesmo depois de o IP atingir seu limite, o token possui seu próprio limite e tem prioridade sobre o limite do IP.

## Teste do token básico

O `token-basic` possui limite de 20 requisições por segundo.

Primeiro, limpe o Redis:

```powershell
docker compose exec redis redis-cli FLUSHDB
```

Depois, defina o token:

```powershell
$headers = @{ API_KEY = "token-basic" }
```

Envie 22 requisições:

```powershell
1..22 | ForEach-Object {
    try {
        $r = Invoke-WebRequest `
            http://localhost:8080/ `
            -Headers $headers `
            -UseBasicParsing

        Write-Host "$_ -> $($r.StatusCode)"
    }
    catch {
        Write-Host "$_ -> $($_.Exception.Response.StatusCode.value__)"
    }
}
```

O resultado esperado é:

```text
1 -> 200
...
20 -> 200
21 -> 429
22 -> 429
```

## Resposta HTTP 429

Quando o limite é ultrapassado, a aplicação retorna:

```text
HTTP 429 Too Many Requests
```

Com a seguinte mensagem:

```text
you have reached the maximum number of requests or actions allowed within a certain time frame
```

Essa mensagem também é validada por teste automatizado.

## Bloqueio

Depois que o limite é excedido, o cliente infrator fica bloqueado pelo período configurado em:

```env
BLOCK_DURATION_SECONDS=300
```

Isso representa:

```text
300 segundos = 5 minutos
```

O período de bloqueio pode ser alterado por meio da variável de ambiente.

Durante o período de bloqueio, novas requisições do IP ou token bloqueado continuam sendo rejeitadas.

## Testes automatizados

Para executar todos os testes utilizando Docker:

```powershell
docker compose run --rm test
```

O projeto possui testes automatizados para validar as principais regras do Rate Limiter.

### `TestTokenHasPrecedenceOverIP`

Verifica que, quando existe um token, o limite configurado para o token possui prioridade sobre o limite do IP.

### `TestIPUsesIPLimit`

Verifica que uma requisição sem token utiliza o limite configurado para o IP.

### `TestReturns429WithExactMessage`

Verifica que, ao exceder o limite, o middleware retorna HTTP 429 e a mensagem exigida exatamente.

### `TestTokenOverridesIP`

Verifica que o token pode continuar realizando requisições de acordo com seu próprio limite, mesmo quando o limite do IP já foi atingido.

A execução dos testes deve terminar com:

```text
PASS
```

e os pacotes de teste devem apresentar resultado semelhante a:

```text
ok
```

## Redis e atomicidade

O Redis é utilizado para armazenar o estado do Rate Limiter, incluindo os contadores e informações necessárias para o controle das requisições e bloqueios.

A operação de controle utiliza **Lua** para manter a atualização do contador de forma atômica.

Isso ajuda a evitar condições de corrida quando várias requisições chegam praticamente ao mesmo tempo.

## Estratégia de persistência

A camada de persistência é desacoplada por meio da interface `RateStore`.

A implementação atual é:

```text
RateStore
    |
    +-- RedisStore
```

O `RateLimiter` trabalha com a interface `RateStore`, e não diretamente com o Redis.

Dessa forma, uma nova estratégia de persistência pode ser adicionada futuramente, por exemplo:

```text
RateStore
    |
    +-- RedisStore
    |
    +-- MemoryStore
```

Para utilizar outra estratégia, basta criar uma nova implementação da interface `RateStore` e configurá-la na aplicação. A lógica principal do Rate Limiter não precisa ser alterada.

## Docker

O projeto possui:

- `Dockerfile` para a aplicação
- `docker-compose.yaml` para orquestração dos serviços
- Serviço da aplicação
- Serviço Redis
- Serviço separado para execução dos testes

O avaliador consegue executar o projeto sem precisar instalar Go ou Redis diretamente na máquina.

### Subir a aplicação

```powershell
docker compose up --build
```

### Executar os testes

```powershell
docker compose run --rm test
```

### Parar os containers

```powershell
docker compose down
```

### Limpar o Redis

```powershell
docker compose exec redis redis-cli FLUSHDB
```

## Executando o projeto do zero

A sequência recomendada é:

### 1. Criar o `.env`

```powershell
Copy-Item .env.example .env
```

### 2. Subir os containers

```powershell
docker compose up --build
```

### 3. Verificar a aplicação

Em outro terminal:

```powershell
Invoke-WebRequest http://localhost:8080/ -UseBasicParsing
```

A resposta esperada é:

```text
rate limiter is running
```

### 4. Executar os testes automatizados

Em outro terminal:

```powershell
docker compose run --rm test
```

### 5. Executar os testes manuais

Depois dos testes automatizados, é possível executar os testes de IP, token, precedência do token e `token-basic` descritos neste README.

## Variáveis de ambiente

Todas as configurações importantes ficam no ambiente da aplicação.

Não é necessário alterar o código para modificar os limites.

Por exemplo:

```env
RATE_LIMIT_IP=10

RATE_LIMIT_TOKEN_DEFAULT=10

TOKEN_LIMITS_JSON={"token-premium":100,"token-basic":20}

BLOCK_DURATION_SECONDS=300
```

## Segurança

O arquivo `.env` não deve ser versionado caso contenha informações sensíveis.

O projeto disponibiliza:

```text
.env.example
```

como modelo de configuração.

O `.env.example` pode ser enviado para o repositório normalmente.

## Resultado

Com este projeto, demonstro a implementação de:

- Rate Limiter desenvolvido em Go
- Middleware HTTP
- Limitação por IP
- Limitação por token
- Token com prioridade sobre IP
- HTTP 429
- Mensagem de erro exata
- Bloqueio configurável
- Persistência com Redis
- Operações atômicas utilizando Lua
- Strategy Pattern para persistência
- Separação entre regra de negócio e middleware
- Configuração por variáveis de ambiente
- Docker e Docker Compose
- Testes automatizados
- Testes manuais documentados

O projeto pode ser executado e testado utilizando apenas **Docker e Docker Compose**, sem a necessidade de instalar Go ou Redis diretamente na máquina do avaliador.
