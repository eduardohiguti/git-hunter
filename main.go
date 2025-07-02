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
	hunterRootDir = ".hunter"
	repoObjectsDir = "objects"
	repoCommitsDir = "commits"
	hunterIndexFile = "index"
)

func getObjectsDirPath() string {
	return filepath.Join(hunterRootDir, repoObjectsDir)
}

func getCommitsDirPath() string {
	return filepath.Join(hunterRootDir, repoCommitsDir)
}

func getIndexPath() string {
	return filepath.Join(hunterRootDir, hunterIndexFile)
}

func initRepo() error {
	fmt.Printf("Inicializando repositório Hunter em '%s'\n", hunterRootDir)

	err := os.MkdirAll(hunterRootDir, 0755)
	if err != nil {
		return fmt.Errorf("erro ao criar diretório raiz do Hunter '%s': %w", hunterRootDir, err)
	}

	err = os.MkdirAll(getObjectsDirPath(), 0755)
	if err != nil {
		return fmt.Errorf("erro ao criar diretório de objetos '%s': %w", getObjectsDirPath(), err)
	}

	err = os.MkdirAll(getCommitsDirPath(), 0755)
	if err != nil {
		return fmt.Errorf("erro ao criar diretório de commits '%s': %w", getCommitsDirPath(), err)
	}

	indexPath := getIndexPath()
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		_, err := os.Create(indexPath)
		if err != nil {
			return fmt.Errorf("erro ao criar arquivo de índice '%s': %w", indexPath, err)
		}
	} else if err != nil {
		return fmt.Errorf("erro ao verificar arquivo de índice '%s': %w", indexPath, err)
	}

	fmt.Println("Repositório Hunter inicializado com sucesso!")
	return nil
}

func checkRepoInitialized() error {
	if _, err := os.Stat(hunterRootDir); os.IsNotExist(err) {
		return fmt.Errorf("repositório Hunter não inicializado. Use 'hunter init' primeiro")
	}
	return nil
}

func addToIndex(filePath string) (string, error) {
	if err := checkRepoInitialized(); err != nil {
		return "", err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("erro ao ler arquivo '%s' no diretório de trabalho: %w", filePath, err)
	}

	hasher := sha1.New()
	hasher.Write(content)
	hash := hex.EncodeToString(hasher.Sum(nil))

	objectPath := filepath.Join(getObjectsDirPath(), hash)
	if _, err := os.Stat(objectPath); os.IsNotExist(err) {
		err = os.WriteFile(objectPath, content, 0644)
		if err != nil {
			return "", fmt.Errorf("erro ao salvar objeto no índice '%s': %w", objectPath,err)
		}
		fmt.Printf("'%s' salvo com hash: %s\n", filePath, hash)
	} else if err != nil {
		return "", fmt.Errorf("erro ao verificar objeto '%s': %w", objectPath,err)
	} else {
		fmt.Printf("'%s' já existe no índice. Não regravado\n", filePath)
	}

	indexPath := getIndexPath()
	indexEntries := make(map[string]string)

	existingIndexContent, err := os.ReadFile(indexPath)
	if err == nil {
		lines := strings.Split(string(existingIndexContent), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				indexEntries[parts[1]] = parts[0]
			}
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("erro ao ler o arquivo do índice '%s': %w", indexPath,err)
	}

	indexEntries[filePath] = hash

	var newIndexContent strings.Builder
	for path, h := range indexEntries {
		newIndexContent.WriteString(fmt.Sprintf("%s %s\n", h, path))
	}

	err = os.WriteFile(indexPath, []byte(newIndexContent.String()), 0644)
	if err != nil {
		return "", fmt.Errorf("erro ao atualizar o índice '%s': %w", indexPath,err)
	}

	fmt.Printf("'%s' adicionado ao índice\n", filePath)
	return hash, nil
}

func commitChanges(message string) error {
	if err := checkRepoInitialized(); err != nil {
		return err
	}

	indexPath := getIndexPath()
	indexContent, err := os.ReadFile(indexPath)
	if os.IsNotExist(err) || len(strings.TrimSpace(string(indexContent))) == 0 {
		return fmt.Errorf("não há arquivos na área de staging para commitar. Use 'hunter add <arquivo>' primeiro")
	}
	if err != nil {
		return fmt.Errorf("erro ao ler o índice '%s': %w", indexPath, err)
	}

	commitId := sha1.New()
	commitId.Write([]byte(message))
	commitId.Write(indexContent)
	commitId.Write([]byte(time.Now().Format(time.RFC3339Nano)))

	commitHash := hex.EncodeToString(commitId.Sum(nil))

	commitContent := fmt.Sprintf(
		"commit %s\n"+
		"date: %s\n"+
		"\n"+
		"	%s\n"+
		"\n"+
		"arquivos staged (do índice):\n%s",
		commitHash,
		time.Now().Format(time.RFC3339),
		message,
		string(indexContent),
	)

	commitPath := filepath.Join(getCommitsDirPath(), commitHash)
	err = os.WriteFile(commitPath, []byte(commitContent), 0644)
	if err != nil {
		return fmt.Errorf("erro ao salvar commit em '%s': %w", commitPath, err)
	}

	fmt.Printf("Commit '%s' criado com sucesso. Hash: %s\n", message, commitHash)

	err = os.WriteFile(indexPath, []byte{}, 0644)
	if err != nil {
		fmt.Printf("Aviso: erro ao limpar o arquivo do índice '%s': %v\n", indexPath, err)
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
		fmt.Println("	init				Inicializa um novo repositório Hunter")
		fmt.Println("	add				Adiciona o arquivo à área de staging")
		fmt.Println("	commit \"<mensagem>\"		Cria um novo commit com os arquivos na área de staging")
		fmt.Println()
		return
	}

	switch args[0] {
	case "oi":
		fmt.Println("oi")
	case "init":
		err := initRepo()
		if err != nil {
			fmt.Println("Erro:", err)
		}
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