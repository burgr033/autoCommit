package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/burgr033/autoCommit/internal/filetypes"
	git "github.com/go-git/go-git/v5"
)

type (
	CommitBody    []Msg
	GroupedCommit map[string][]string
	Msg           struct {
		Conventional string
		GitStatus    string
		File         string
		Extra        string
	}
)

// Cache for conventional type lookups
var (
	conventionalTypeCache = make(map[string]string)
	cacheMutex            sync.RWMutex
)

func (b *CommitBody) toString() string {
	var bodyString []string
	bodyString = append(bodyString, "# This is an automated commit message")
	bodyString = append(bodyString, "")

	// Group similar messages together
	grouped := b.groupMessages()
	for key, files := range grouped {
		bodyString = append(bodyString, fmt.Sprintf("# %s %s", key, strings.Join(files, ", ")))
	}

	bodyString = append(bodyString, "")
	bodyString = append(bodyString, "# This is the Footer of the automated commit message")
	return strings.Join(bodyString, "\n")
}

func (b *CommitBody) groupMessages() GroupedCommit {
	grouped := make(GroupedCommit)
	for _, msg := range *b {
		key := fmt.Sprintf("%s: %s", msg.Conventional, msg.GitStatus)
		grouped[key] = append(grouped[key], msg.File)
	}
	return grouped
}

func (m *Msg) toString() string {
	return fmt.Sprintf("# %s: %s %s %s", m.Conventional, m.GitStatus, m.File, m.Extra)
}

// getConventionalType with caching
func getConventionalType(filename string) string {
	lowerFilename := strings.ToLower(filename)

	// Check cache first
	cacheMutex.RLock()
	if commitType, exists := conventionalTypeCache[lowerFilename]; exists {
		cacheMutex.RUnlock()
		return commitType
	}
	cacheMutex.RUnlock()

	var commitType string

	// Exact filename match
	if commitType, exists := filetypes.NameMapping[lowerFilename]; exists {
		cacheMutex.Lock()
		conventionalTypeCache[lowerFilename] = commitType
		cacheMutex.Unlock()
		return commitType
	}

	// Directory wildcard match
	for pattern, cType := range filetypes.NameMapping {
		if strings.HasSuffix(pattern, "/*") {
			dir := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(lowerFilename, dir+"/") {
				commitType = cType
				break
			}
		}
	}

	// Extension wildcard match if no directory match found
	if commitType == "" {
		base := filepath.Base(lowerFilename)
		for pattern, cType := range filetypes.NameMapping {
			if strings.HasPrefix(pattern, "*") {
				if match, _ := filepath.Match(pattern, base); match {
					commitType = cType
					break
				}
			}
		}
	}

	// Default to unknown
	if commitType == "" {
		commitType = filetypes.ConventionalUnknown
	}

	// Store in cache
	cacheMutex.Lock()
	conventionalTypeCache[lowerFilename] = commitType
	cacheMutex.Unlock()

	return commitType
}

func getNamingOfBranch(branch string) string {
	if commitType, exists := filetypes.BranchMapping[branch]; exists {
		return commitType
	}
	return filetypes.ConventionalUnknown
}

func getGitStatusText(gs git.StatusCode) string {
	switch gs {
	case git.Modified:
		return "modified"
	case git.Added:
		return "added"
	case git.Deleted:
		return "deleted"
	case git.Renamed:
		return "renamed"
	case git.Copied:
		return "copied"
	default:
		return ""
	}
}

func determineGitStatus(repo *git.Repository) CommitBody {
	// Get all Git info in one block to reduce redundant operations
	wt, err := repo.Worktree()
	if err != nil {
		log.Fatalf("Failed to get worktree: %v", err)
	}

	status, err := wt.Status()
	if err != nil {
		log.Fatalf("Failed to get status from worktree: %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		log.Fatalf("Failed to get HEAD: %v", err)
	}

	branchName := head.Name().Short()
	branchType := getNamingOfBranch(branchName)

	var messages []Msg
	for file, statusEntry := range status {
		// Skip if no staging status
		if statusEntry.Staging == git.Untracked {
			continue
		}

		gitStatusText := getGitStatusText(statusEntry.Staging)
		if gitStatusText == "" {
			continue
		}

		var message Msg
		message.File = file
		message.GitStatus = gitStatusText

		// Get conventional type with caching
		message.Conventional = getConventionalType(file)
		if message.Conventional == filetypes.ConventionalUnknown {
			message.Conventional = branchType
		}

		messages = append(messages, message)
	}

	return messages
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

	statusString := determineGitStatus(repo)

	err = os.WriteFile(commitMsgFile, []byte(statusString.toString()), 0o644)
	if err != nil {
		log.Fatalf("Error writing to commit message file: %v", err)
	}
}
