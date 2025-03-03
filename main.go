package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/burgr033/autoCommit/internal/filetypes"
	git "github.com/go-git/go-git/v5"
)

// TODO: Documentation
// TODO: Grouping of output messages (feat: modified FileX, FileY and FileZ)
// TODO: Performance Imporevements (don't know if this stuff cycles to much)

func getConventionalType(filename string) string {
	if commitType, exists := filetypes.NameMapping[filename]; exists {
		return commitType
	}

	for pattern, commitType := range filetypes.NameMapping {
		if strings.HasSuffix(pattern, "/*") {
			dir := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(filename, dir+"/") {
				return commitType
			}
		}
	}

	base := filepath.Base(filename)
	for pattern, commitType := range filetypes.NameMapping {
		if strings.HasPrefix(pattern, "*") {
			if match, _ := filepath.Match(pattern, base); match {
				return commitType
			}
		}
	}

	return filetypes.ConventionalUnkown
}

func getNamingOfBranch(branch string) string {
	if strings.HasPrefix(branch, "feature") {
		return filetypes.ConventionalFeat
	}
	if strings.HasPrefix(branch, "bugfix") {
		return filetypes.ConventionalFix
	}
	if strings.HasPrefix(branch, "release") {
		return filetypes.ConventionalChore
	}
	if strings.HasPrefix(branch, "hotfix") {
		return filetypes.ConventionalFix
	}
	if strings.HasPrefix(branch, "support") {
		return filetypes.ConventionalFix
	}

	return filetypes.ConventionalUnkown
}

func dissectGitStatus(repo *git.Repository) string {
	head, _ := repo.Head()
	var messages []string
	status := getGitStatus(repo)
	for file, statusEntry := range status {
		commitType := getConventionalType(strings.ToLower(file))
		if commitType == filetypes.ConventionalUnkown {
			commitType = getNamingOfBranch(head.Name().Short())
		}
		var message string

		switch statusEntry.Staging {
		case git.Modified:
			message = fmt.Sprintf("%s: modified %s", commitType, file)
		case git.Added:
			message = fmt.Sprintf("%s: added %s", commitType, file)
		case git.Deleted:
			message = fmt.Sprintf("%s: deleted %s", commitType, file)
		case git.Renamed:
			message = fmt.Sprintf("%s: renamed %s", commitType, file)
		case git.Copied:
			message = fmt.Sprintf("%s: copied %s", commitType, file)
		default:
			continue
		}
		message = "# " + message
		messages = append(messages, message)
	}

	return strings.Join(messages, "\n")
}

func getGitStatus(repo *git.Repository) git.Status {
	wt, err := repo.Worktree()
	if err != nil {
		log.Fatalf("Failed to get worktree: %v", err)
	}

	status, err := wt.Status()
	if err != nil {
		log.Fatalf("Failed to get status from worktree: %v", err)
	}

	return status
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <commit-msg-file>")
	}

	commitMsgFile := os.Args[1]

	repo, err := git.PlainOpen(".")
	if err != nil {
		log.Fatalf("Not a git repository: %v", err)
	}

	statusString := dissectGitStatus(repo)

	fmt.Println(statusString)

	err = os.WriteFile(commitMsgFile, []byte(statusString), 0o644)
	if err != nil {
		log.Fatalf("Error writing to commit message file: %v", err)
	}
}
