# Rate Limiter em Go com Redis

Este projeto implementa um **Rate Limiter em Go** como middleware para uma aplicação HTTP, utilizando **Redis** para controlar e persistir os limites de requisições.

A ideia principal é limitar o número de requisições por segundo com base no **IP** ou em um **token enviado no header `API_KEY`**.

## Regras implementadas

- Limite por IP: **10 requisições por segundo**
- Limite padrão por token: **10 requisições por segundo**
- `token-basic`: **20 requisições por segundo**
- `token-premium`: **100 requisições por segundo**
- O token sempre tem prioridade sobre o IP
- Ao ultrapassar o limite, a API retorna **HTTP 429**
- O bloqueio possui duração configurável
- Redis é utilizado para persistência
- A persistência foi desacoplada usando uma interface, permitindo trocar o Redis por outra implementação
- As configurações são feitas por variáveis de ambiente
- O projeto pode ser executado e testado utilizando Docker e Docker Compose

## Tecnologias

- Go
- Redis
- Docker
- Docker Compose
- HTTP Middleware
- Lua para operação atômica no Redis
- Testes automatizados com `go test`

## Arquitetura

O fluxo principal é:

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

A regra de negócio não fica diretamente dentro do middleware.

A interface `RateStore` permite trocar a implementação de persistência no futuro sem precisar alterar a lógica principal do Rate Limiter.

Atualmente a implementação utilizada é `RedisStore`.

## Estrutura

Uma visão simplificada do projeto:

```text
rate-limiter-go/
├── cmd/
│   └── server/
├── internal/
│   ├── config/
│   ├── limiter/
│   ├── middleware/
│   └── store/
├── tests/
├── Dockerfile
├── docker-compose.yaml
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

## Configuração

As configurações ficam no arquivo `.env`.

O projeto possui um `.env.example` para facilitar a configuração.

### Criar o `.env`

No PowerShell:

```powershell
Copy-Item .env.example .env
```

No Linux/macOS:

```bash
cp .env.example .env
```

Exemplo de configuração:

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

| Variável | Descrição |
|---|---|
| `APP_PORT` | Porta da aplicação |
| `RATE_LIMIT_IP` | Limite de requisições por IP |
| `RATE_LIMIT_TOKEN_DEFAULT` | Limite padrão para tokens |
| `TOKEN_LIMITS_JSON` | Limites específicos por token |
| `BLOCK_DURATION_SECONDS` | Tempo de bloqueio |
| `REDIS_ADDR` | Endereço do Redis |
| `REDIS_PASSWORD` | Senha do Redis |
| `REDIS_DB` | Banco utilizado no Redis |
| `RATE_LIMIT_WINDOW_SECONDS` | Janela de controle |

## Executando com Docker

O projeto foi preparado para funcionar utilizando Docker Compose.

### Subir a aplicação

```powershell
docker compose up --build
```

A aplicação ficará disponível em:

```text
http://localhost:8080/
```

Para verificar se está funcionando:

```powershell
Invoke-WebRequest http://localhost:8080/ -UseBasicParsing
```

A resposta esperada é:

```text
rate limiter is running
```

## Teste manual por IP

Primeiro limpo o Redis para começar o teste do zero:

```powershell
docker compose exec redis redis-cli FLUSHDB
```

Depois envio 12 requisições sem o header `API_KEY`:

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

Para testar o `token-premium`:

```powershell
docker compose exec redis redis-cli FLUSHDB
```

Depois:

```powershell
$headers = @{ API_KEY = "token-premium" }

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

## Teste Token > IP

Esse teste demonstra que o token tem prioridade sobre o IP.

Primeiro limpo o Redis:

```powershell
docker compose exec redis redis-cli FLUSHDB
```

Agora faço 12 requisições sem token:

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

As primeiras 10 devem retornar `200` e as duas últimas `429`.

**Não limpe o Redis novamente.**

Agora faço outras 12 requisições utilizando:

```powershell
$headers = @{ API_KEY = "token-premium" }
```

O resultado esperado é:

```text
1 -> 200
2 -> 200
...
12 -> 200
```

Isso demonstra que, mesmo depois de o IP atingir seu limite, o token possui seu próprio limite e tem prioridade.

## Teste do token básico

O `token-basic` possui limite de 20 requisições por segundo.

Limpo o Redis:

```powershell
docker compose exec redis redis-cli FLUSHDB
```

Depois:

```powershell
$headers = @{ API_KEY = "token-basic" }

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

Resultado esperado:

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

Com a mensagem:

```text
you have reached the maximum number of requests or actions allowed within a certain time frame
```

A mensagem é validada também por teste automatizado.

## Bloqueio

Depois que o limite é excedido, o cliente fica bloqueado pelo período configurado em:

```env
BLOCK_DURATION_SECONDS=300
```

Isso representa:

```text
300 segundos = 5 minutos
```

O tempo pode ser alterado no `.env`.

## Testes automatizados

Para executar todos os testes:

```powershell
docker compose run --rm test
```

O projeto possui testes para validar principalmente:

### `TestTokenHasPrecedenceOverIP`

Verifica que, quando existe um token, o limite do token tem prioridade sobre o limite do IP.

### `TestIPUsesIPLimit`

Verifica que uma requisição sem token utiliza o limite configurado para o IP.

### `TestReturns429WithExactMessage`

Verifica que, ao exceder o limite, o middleware retorna HTTP 429 e a mensagem exigida exatamente.

### `TestTokenOverridesIP`

Verifica que o token pode continuar realizando requisições mesmo quando o limite do IP já foi atingido.

A execução deve terminar com:

```text
PASS
```

e os pacotes de teste devem aparecer como:

```text
ok
```

## Redis e atomicidade

O Redis é utilizado para armazenar o estado do Rate Limiter.

A operação de controle utiliza Lua para manter a atualização do contador de forma atômica.

Isso evita problemas de concorrência quando várias requisições chegam praticamente ao mesmo tempo.

## Estratégia de persistência

A persistência é desacoplada através da interface `RateStore`.

A implementação atual é:

```text
RateStore
    |
    +-- RedisStore
```

Isso permite substituir o Redis futuramente por outra estratégia de armazenamento sem alterar a regra principal do Rate Limiter.

## Docker

A aplicação possui:

- `Dockerfile`
- `docker-compose.yaml`
- serviço da aplicação
- serviço Redis
- serviço separado para executar os testes

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

## Rodando os testes do zero

Minha sequência recomendada é:

### 1. Criar o `.env`

```powershell
Copy-Item .env.example .env
```

### 2. Subir os containers

```powershell
docker compose up --build
```

### 3. Verificar a aplicação

```powershell
Invoke-WebRequest http://localhost:8080/ -UseBasicParsing
```

### 4. Executar os testes automatizados

Em outro terminal:

```powershell
docker compose run --rm test
```

### 5. Testar manualmente

Depois dos testes automatizados, posso executar os testes de IP, token, prioridade do token e `token-basic` descritos neste README.

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

Com este projeto eu consigo demonstrar:

- Rate Limiter desenvolvido em Go
- Middleware HTTP
- Limitação por IP
- Limitação por token
- Token com prioridade sobre IP
- HTTP 429
- Mensagem de erro exata
- Bloqueio configurável
- Persistência com Redis
- Operação atômica utilizando Lua
- Strategy Pattern para persistência
- Separação entre regra de negócio e middleware
- Configuração por variáveis de ambiente
- Docker e Docker Compose
- Testes automatizados
- Testes manuais documentados

O projeto pode ser executado e testado utilizando apenas Docker e Docker Compose.
