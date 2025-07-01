package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"strings"
)

const (
	repoObjectsDir = "repo_objects"
	repoCommitsDir = "repo_commits"
	currentSnapshotFile = ".hunter_snapshot"
)

func addToRepository(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("erro ao ler arquivo no diretório de trabalho: %w", err)
	}
	
	hasher := sha1.New()
	hasher.Write(content)
	hash := hex.EncodeToString(hasher.Sum(nil))

	repoObjectsDir := "repo_objects"
	err = os.MkdirAll(repoObjectsDir, 0755)
	if err != nil {
		return "", fmt.Errorf("erro ao criar diretório do repositório: %w", err)
	}

	objectPath := filepath.Join(repoObjectsDir, hash) 
	err = os.WriteFile(objectPath, content, 0644)
	if err != nil {
		return "", fmt.Errorf("erro ao salvar objeto no repositório: %w", err)
	}

	fmt.Printf("Conteúdo de '%s' adicionado ao 'repositório' com hash: %s\n", filePath, hash)

	snapshotEntry := fmt.Sprintf("%s %s\n", hash, filePath)
	f, err := os.OpenFile(currentSnapshotFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("erro ao abrir arquivo de snapshot: %w", err)
	}
	defer f.Close()	
	if _, err := f.WriteString(snapshotEntry); err != nil {
		return "", fmt.Errorf("erro ao escrever no arquivo de snapshot: %w", err)
	}

	return hash, nil
}

func commitChanges(message string) error {
	snapshotContent, err := os.ReadFile(currentSnapshotFile)
	if os.IsNotExist(err) || len(snapshotContent) == 0 {
		return fmt.Errorf("não há arquivos para comitar. Use 'hunter add <arquivo>' primeiro.")
	}
	if err != nil {
		return fmt.Errorf("erro ao ler snapshot atual: %w", err)
	}

	commitId := sha1.New()
	commitId.Write([]byte(message))
	commitId.Write(snapshotContent)

	commitId.Write([]byte(time.Now().Format(time.RFC3339Nano)))

	commitHash := hex.EncodeToString(commitId.Sum(nil))

	commitContent := fmt.Sprintf(
		"commit %s\n"+
		"data: %s\n"+
		"\n"+
		"	%s\n"+
		"\n"+
		"Arquivos:\n%s",
		commitHash,
		time.Now().Format(time.RFC3339Nano),
		message,
		string(snapshotContent),
	)

	err = os.MkdirAll(repoCommitsDir, 0755)
	if err != nil {
		return fmt.Errorf("erro ao criar diretório de commits: %w", err)
	}

	commitPath := filepath.Join(repoCommitsDir, commitHash)
	err = os.WriteFile(commitPath, []byte(commitContent), 0644)
	if err != nil {
		return fmt.Errorf("erro ao salvar objeto commmit: %w", err)
	}

	fmt.Printf("Commit '%s' criado com sucesso! Hash: %s\n", message, commitHash)

	err = os.WriteFile(currentSnapshotFile, []byte{}, 0644)
	if err != nil {
		fmt.Printf("Aviso: erro ao limpar arquivo de snapshot: %v\n", err)
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
		fmt.Println("	commit		Cria um novo commit com os arquivos no 'snapshot'")
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
		_, err := addToRepository(filePath)
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