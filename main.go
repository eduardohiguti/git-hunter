package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	hunterRootDir   = ".hunter"
	repoObjectsDir  = "objects"
	repoCommitsDir  = "commits"
	hunterIndexFile = "index"
)

var repoRoot string

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, hunterRootDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Abs(dir)
		}
		master := filepath.Dir(dir)
		if master == dir {
			break
		}
		dir = master
	}
	return "", fmt.Errorf("repositório Hunter não inicializado. Use 'hunter init' primeiro")
}

func getRepoPath(sub ...string) string {
	return filepath.Join(append([]string{repoRoot}, sub...)...)
}

func getObjectsDirPath() string {
	return getRepoPath(hunterRootDir, repoObjectsDir)
}

func getCommitsDirPath() string {
	return getRepoPath(hunterRootDir, repoCommitsDir)
}

func getIndexPath() string {
	return getRepoPath(hunterRootDir, hunterIndexFile)
}

func getHeadPath() string {
	return getRepoPath(hunterRootDir, "HEAD")
}

func getHeadCommitHash() (string, error) {
	content, err := os.ReadFile(getHeadPath())
	if os.IsNotExist((err)) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("erro ao ler o HEAD: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}

func updateHeadCommitHash(hash string) error {
	return os.WriteFile(getHeadPath(), []byte(hash), 0644)
}

func getIndexFiles() (map[string]string, error) {
	indexEntries := make(map[string]string)
	indexPath := getIndexPath()
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return indexEntries, nil
		}
		return nil, err
	}

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			indexEntries[parts[1]] = parts[0]
		}
	}

	return indexEntries, nil

}

func getCommitFiles(commitHash string) (map[string]string, error) {
	commitFiles := make(map[string]string)
	if commitHash == "" {
		return commitFiles, nil
	}

	commitPath := filepath.Join(getCommitsDirPath(), commitHash)
	data, err := os.ReadFile(commitPath)
	if err != nil {
		return nil, fmt.Errorf("não foi possível ler o commit %s: %w", commitHash, err)
	}

	content := string(data)
	fileSectionIndex := strings.Index(content, "arquivos staged (do índice):\n")
	if fileSectionIndex == -1 {
		return commitFiles, nil
	}

	fileSection := content[fileSectionIndex+len("arquivos staged (do índice):\n"):]
	for _, line := range strings.Split(strings.TrimSpace(fileSection), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			commitFiles[parts[1]] = parts[0]
		}
	}

	return commitFiles, nil
} 

func initRepo() error {
	cwd, _ := os.Getwd()
	repoRoot = cwd

	hunterRoot := filepath.Join(repoRoot, hunterRootDir)
	if err := os.MkdirAll(filepath.Join(hunterRoot, repoObjectsDir), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(hunterRoot, repoCommitsDir), 0755); err != nil {
		return err
	}
	indexPath := filepath.Join(hunterRoot, hunterIndexFile)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		f, err := os.Create(indexPath)
		if err != nil {
			return err
		}
		f.Close()
	}

	absPath, _ := filepath.Abs(hunterRoot)
	fmt.Printf("Repositório Hunter inicializado em %s\n", absPath)
	return nil
}

func ensureRepo() error {
	if repoRoot != "" {
		return nil
	}
	root, err := findRepoRoot()
	if err != nil {
		return err
	}
	repoRoot = root
	return nil
}

func hashAndSaveObject(absPath string) (string, error) {
    content, err := os.ReadFile(absPath)
    if err != nil {
        return "", err
    }
    hasher := sha1.New()
    hasher.Write(content)
    hash := hex.EncodeToString(hasher.Sum(nil))

    objectPath := filepath.Join(getObjectsDirPath(), hash)
    if _, err := os.Stat(objectPath); errors.Is(err, fs.ErrNotExist) {
        fd, err := os.OpenFile(objectPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
        if err != nil && !errors.Is(err, fs.ErrExist) {
            return "", err
        }
        if err == nil {
            _, werr := fd.Write(content)
            fd.Close()
            if werr != nil {
                return "", werr
            }
        }
    } else if err != nil {
        return "", err
    }

    return hash, nil
}

func addToIndex(filePath string) (string, error) {
	if err := ensureRepo(); err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return "", err
	}
	rel = filepath.Clean(rel)
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("arquivo fora do repositório")
	}

	hash, err := hashAndSaveObject(absPath)
	if err != nil {
		return "", err
	}
	fmt.Printf("%s salvo com hash: %s\n", rel, hash)

	indexEntries := make(map[string]string)
	indexPath := getIndexPath()
	if data, err := os.ReadFile(indexPath); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "/t", 2)
			if len(parts) == 2 {
				indexEntries[parts[1]] = parts[0]
			}
		}
	}

	indexEntries[rel] = hash

	paths := make([]string, 0, len(indexEntries))
	for p := range indexEntries {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var b strings.Builder
	for _, p := range paths {
		b.WriteString(fmt.Sprintf("%s\t%s\n", indexEntries[p], p))
	}

	if err := os.WriteFile(indexPath, []byte(b.String()), 0644); err != nil {
		return "", err
	}

	fmt.Printf("'%s' adicionado ao índice\n", rel)
	return hash, nil
}

func commitChanges(message string) error {
	if err := ensureRepo(); err != nil {
		return err
	}

	indexPath := getIndexPath()
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || len(strings.TrimSpace(string(indexContent))) == 0 {
			return fmt.Errorf("não há arquivos na área de staging para commitar. Use 'hunter add <arquivo>' primeiro")
		}
		return err
	}

	parentHash, _ := getHeadCommitHash()
	commitTimestamp := time.Now().Format(time.RFC3339)

	var tempCommitContent strings.Builder
	tempCommitContent.WriteString(fmt.Sprintf("mestre: %s\n", parentHash))
	tempCommitContent.WriteString(fmt.Sprintf("data: %s\n\n", commitTimestamp))
	tempCommitContent.WriteString(fmt.Sprintf("%s\n\n", message))
	tempCommitContent.WriteString("arquivos staged (do índice):\n")
	tempCommitContent.Write(indexContent)

	hasher := sha1.New()
	hasher.Write([]byte(tempCommitContent.String()))
	commitHash := hex.EncodeToString(hasher.Sum(nil))

	finalCommitContent := fmt.Sprintf("commit %s\n%s", commitHash, tempCommitContent.String())
	commitPath := filepath.Join(getCommitsDirPath(), commitHash)

	if err := os.WriteFile(commitPath, []byte(finalCommitContent), 0644); err != nil {
		return err
	}
	if err := updateHeadCommitHash(commitHash); err != nil {
		return err
	}
	fmt.Printf("Commit '%s' criado com sucesso. Hash: %s\n", message, commitHash)

	os.WriteFile(indexPath, []byte{}, 0644)
	return nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil	
}

func showStatus() error {
	if err := ensureRepo(); err != nil {
		return err
	}

	headHash, err := getHeadCommitHash()
	if err != nil {
		return err
	}

	headFiles, err := getCommitFiles(headHash)
	if err != nil {
		return err
	}

	indexFiles, err := getIndexFiles()
	if err != nil {
		return err
	}

	var stagedNew, stagedModified, stagedDeleted []string
	var unstagedModified, unstagedDeleted []string
	var untracked []string

	allPaths := make(map[string]bool)
	for path := range headFiles {
		allPaths[path] = true
	}
	for path := range indexFiles {
		allPaths[path] = true
	}

	err = filepath.Walk(repoRoot, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}
		allPaths[rel] = true
		return nil
	})
	if err != nil {
		return err
	}

	sortedPaths := make([]string, 0, len(allPaths))
	for path := range allPaths {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)

	for _, path := range sortedPaths {
		headHash, inHead := headFiles[path]
		indexHash, inIndex := indexFiles[path]

		absPath := getRepoPath(path)
		_, statErr := os.Stat(absPath)
		fileInWorkDir := !os.IsNotExist(statErr)

		var workDirHash string 
		if fileInWorkDir {
			workDirHash, err = fileHash(absPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao calcular hash de %s: %v\n", absPath, err)
			}
		}

		if inHead != inIndex || headHash != indexHash {
			if !inHead && inIndex {
				stagedNew = append(stagedNew, path)
			} else if inHead && !inIndex {
				stagedDeleted = append(stagedDeleted, path)
			} else {
				stagedModified = append(stagedModified, path)
			}
		}

		if inIndex {
			if !fileInWorkDir {
				unstagedDeleted = append(unstagedDeleted, path)
			} else if indexHash != workDirHash {
				unstagedModified = append(unstagedModified, path)
			}
		} else if fileInWorkDir {
			untracked = append(untracked, path)
		}
	}

	fmt.Println("Status do repositório Hunter:")

	if len(stagedNew) == 0 && len(stagedModified) == 0 && len(stagedDeleted) == 0 {
		fmt.Println("\nNenhuma mudança na área de staging")
	} else {
		fmt.Println("\nMudanças a serem commitadas:")
		for _, f := range stagedNew {
			fmt.Printf("	%-12s %s\n", "novo arquivo:", f)
		}
		for _, f := range stagedModified {
			fmt.Printf("	%-12s %s\n", "modificado:", f)
		}
		for _, f := range stagedDeleted {
			fmt.Printf("	%-12s %s\n", "deletado:", f)
		}
	}

	if len(unstagedModified) > 0 || len(unstagedDeleted) > 0 {
		fmt.Println("\nMudanças não preparadas para commit:")
		fmt.Println("	(use \"hunter add <arquivo>...\" para preparar para commit)")
		for _, f := range unstagedModified {
			fmt.Printf("	%-12s %s\n", "modificado:", f)
		}
		for _, f := range unstagedDeleted {
			fmt.Printf("	%-12s %s\n", "deletado:", f)
		}
	}

	if len(untracked) > 0 {
		fmt.Println("\nArquivos não rastreados:")
		fmt.Println("	(use \"hunter add <arquivo>...\" para incluir no que será commitado)")
		for _, f := range untracked {
			fmt.Printf("	%s\n", f)
		}
	}

	if len(stagedNew) == 0 && len(stagedModified) == 0 && len(stagedDeleted) == 0 && len(unstagedModified) == 0 && len(unstagedDeleted) == 0 && len(untracked) == 0 {
		fmt.Println("\nNada a commitar, diretório de trabalho limpo")
	}

	return nil
}

func menu() {
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
		fmt.Println("	commit -m \"<mensagem>\"		Cria um novo commit com os arquivos na área de staging")
		fmt.Println("	status				Mostra o status do repositório")
		fmt.Println()
}

func main() {
	args := os.Args[1:]
	commitCmd := flag.NewFlagSet("commit", flag.ExitOnError)
	commitMessage := commitCmd.String("m", "", "Mensagem do commit (obrigatória)")

	if len(args) == 0 {
		menu()
		return
	}

	switch args[0] {
	case "oi":
		fmt.Println("oi")
	case "init":
		if err := initRepo(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	case "add":
		if len(args) < 2 {
			fmt.Println("Uso: add <caminho-do-arquivo>")
			return
		}
		if _, err := addToIndex(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	case "commit":
		commitCmd.Parse(args[1:])
		if *commitMessage == "" {
			fmt.Println("Erro: a flag -m com a mensagem do commit é obrigatória")
			fmt.Println("Uso: hunter commit -m \"sua mensagem\"")
			return
		}
		if err := commitChanges(*commitMessage); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := showStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Comando não reconhecido:", args[0])
	}
}