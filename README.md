# Hunter

Hunter é uma ferramenta de linha de comando para versionamento de arquivos, inspirada no Git, escrita em Go. É um projeto simples para demonstrar os conceitos básicos de um sistema de controle de versão.

## Funcionalidades

  * **Inicializar um repositório:** Cria a estrutura de diretórios necessária (`.hunter`) para começar a versionar os arquivos.
  * **Adicionar arquivos:** Adiciona arquivos a uma "Staging Area" (índice), preparando-os para o próximo commit.
  * **Commits:** Salva o estado atual dos arquivos na Staging Area como um novo ponto no histórico.

-----

## Como Funciona

O Hunter cria um diretório oculto chamado `.hunter` na raiz do seu projeto. Dentro dele:

  * **`objects/`**: Armazena o conteúdo de cada arquivo adicionado como um "blob". O nome de cada blob é o hash SHA-1 do seu conteúdo, evitando duplicação de dados.
  * **`commits/`**: Armazena os objetos de commit. Cada commit contém uma mensagem, data e uma referência ao estado dos arquivos naquele momento.
  * **`index`**: É a "Staging Area". Um arquivo de texto simples que mapeia os nomes dos arquivos que serão incluídos no próximo commit aos seus respectivos hashes de conteúdo na pasta `objects`.

-----

## Uso

Primeiro, compile o projeto para gerar o executável `hunter`.

```bash
go build -o hunter
```

### 1\. Inicializar o Repositório

Para começar a usar o Hunter em um projeto, execute o comando `init`. Isso criará o diretório `.hunter` no seu diretório atual.

```bash
./hunter init
```

### 2\. Adicionar Arquivos

Para adicionar um arquivo (ou atualizar suas modificações) à Staging Area, use o comando `add`.

```bash
# Adiciona um arquivo chamado 'meu_arquivo.txt'
./hunter add meu_arquivo.txt
```

### 3\. Fazer um Commit

Após adicionar os arquivos desejados, você pode criar um "commit" para salvar permanentemente essas mudanças no histórico. É obrigatório fornecer uma mensagem descritiva para o commit.

```bash
./hunter commit "Este é o meu primeiro commit"
```

Isso criará um novo objeto de commit e limpará a Staging Area para o próximo conjunto de alterações.