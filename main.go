package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	repoObjectsDir = "repo_objects"
	repoCommitsDir = "repo_commits"
	hunterIndexFile = ".hunter_index"
)

func addToIndex(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("erro ao ler arquivo '%s' no diretório de trabalho: %w", filePath, err)
	}

	hasher := sha1.New()
	hasher.Write(content)
	hash := hex.EncodeToString(hasher.Sum(nil))

	err = os.MkdirAll(repoObjectsDir, 0755)
	if err != nil {
		return "", fmt.Errorf("erro ao criar diretório de objetos do repositório: %w", err)
	}

	objectPath := filepath.Join(repoObjectsDir, hash)
	if _, err := os.Stat(objectPath); os.IsNotExist(err) {
		err = os.WriteFile(objectPath, content, 0644)
		if err != nil {
			return "", fmt.Errorf("erro ao salvar objeto blob no repositório: %w", err)
		}
		fmt.Printf("Conteúdo do arquivo '%s' salvo como blob com hash: %s\n", filePath, hash)
	} else if err != nil {
		return "", fmt.Errorf("erro ao verificar objeto blob: %w", err)
	} else {
		fmt.Printf("Blob para '%s' (hash: %s) já existe no repositório. Não regravado\n", filePath, hash)
	}

	indexEntries := make(map[string]string)

	existingIndexContent, err := os.ReadFile(hunterIndexFile)
	if err == nil {
		lines := strings.Split(string(existingIndexContent), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				indexEntries[parts[1]] = parts[0]
			}
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("erro ao ler o arquivo de índice: %w", err)
	}

	indexEntries[filePath] = hash
	var newIndexContent strings.Builder
	for path, h := range indexEntries {
		newIndexContent.WriteString(fmt.Sprintf("%s %s\n", h, path))
	}

	err = os.WriteFile(hunterIndexFile, []byte(newIndexContent.String()), 0644)
	if err != nil {
		return "", fmt.Errorf("erro ao atualizar o índice: %w", err)
	}

	fmt.Printf("Arquivo '%s' (hash: %s) adicionado/atualizado no índice\n", filePath, hash)
	return hash, nil
}

func commitChanges(message string) error {
	indexContent, err := os.ReadFile(hunterIndexFile)
	if os.IsNotExist(err) || len(strings.TrimSpace(string(indexContent))) == 0 {
		return fmt.Errorf("não há arquivos na área de staging para commitar. Use 'hunter add <arquivo>' primeiro")
	}
	if err != nil {
		return fmt.Errorf("erro ao ler o índice: %w", err)
	}

	commitId := sha1.New()
	commitId.Write([]byte(message))
	commitId.Write(indexContent)
	commitId.Write([]byte(time.Now().Format(time.RFC3339Nano)))

	commitHash := hex.EncodeToString(commitId.Sum(nil))

	commitContent := fmt.Sprintf(
		"commit %s\n"+
		"Date: %s\n"+
		"\n"+
		"	%s\n"+
		"\n"+
		"Staged Files (from index):\n%s",
		commitHash,
		time.Now().Format(time.RFC3339),
		message,
		string(indexContent),
	)

	err = os.MkdirAll(repoCommitsDir, 0755)
	if err != nil {
		return fmt.Errorf("erro ao criar diretório de commits: %w", err)
	}

	commitPath := filepath.Join(repoCommitsDir, commitHash)
	err = os.WriteFile(commitPath, []byte(commitContent), 0644)
	if err != nil {
		return fmt.Errorf("erro ao salvar objeto commit: %w", err)
	}

	fmt.Printf("Commit '%s' criado com sucesso. Hash: %s\n", message, commitHash)

	err = os.WriteFile(hunterIndexFile, []byte{}, 0644)
	if err != nil {
		fmt.Printf("Aviso: erro ao limpar o arquivo de índice: %v\n", err)
	}

	return nil
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("Hunter é uma ferramenta para versionamento de arquivos")
		fmt.Println()
		fmt.Println("Uso:")
		fmt.Println()
		fmt.Println("	hunter <comando> [argumentos]")
		fmt.Println()
		fmt.Println("Os comandos são:")
		fmt.Println()
		fmt.Println("	add		Adiciona o arquivo para o repositório")
		fmt.Println("	commit		Cria um novo commit com os arquivos na área de staging")
		fmt.Println()
		return
	}

	switch args[0] {
	case "oi":
		fmt.Println("oi")
	case "add":
		if len(args) < 2 {
			fmt.Println("Uso: add <caminho-do-arquivo>")
			return
		}
		filePath := args[1]
		_, err := addToIndex(filePath)
		if err != nil {
			fmt.Println("Erro:", err)
		}
	case "commit":
		if len(args) < 2 {
			fmt.Println("Uso: commit \"<mensagem-do-commit>\"")
			return
		}
		commitMessage := strings.Join(args[1:], " ") 
		err := commitChanges(commitMessage)
		if err != nil {
			fmt.Println("Erro:", err)
		}
 	default: 
		fmt.Println("Comando não reconhecido:", args[0])
	}
}