package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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

	return hash, nil
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("Hunter versionamento de arquivos")
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
	default: 
		fmt.Println("Comando não reconhecido:", args[0])
	}
}